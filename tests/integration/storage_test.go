//go:build integration

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/jullss/banners-rotation/internal/domain"
	"github.com/jullss/banners-rotation/internal/storage/postgres"
)

type StorageSuite struct {
	suite.Suite
	container *tcpostgres.PostgresContainer
	store     *postgres.Storage
}

func TestStorageSuite(t *testing.T) {
	suite.Run(t, new(StorageSuite))
}

func (s *StorageSuite) SetupSuite() {
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("banners"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		tcpostgres.WithInitScripts(migrationsPath()),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2)),
	)
	s.Require().NoError(err)
	s.container = container

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	s.Require().NoError(err)

	store, err := postgres.New(dsn)
	s.Require().NoError(err)
	s.store = store
}

func (s *StorageSuite) TearDownSuite() {
	if s.store != nil {
		s.store.Close()
	}
	if s.container != nil {
		s.container.Terminate(context.Background()) //nolint:errcheck
	}
}

func (s *StorageSuite) TearDownTest() {
	db := s.store.DB()
	_, err := db.ExecContext(context.Background(),
		`TRUNCATE stats, slot_banners, banners, slots, social_groups RESTART IDENTITY CASCADE`,
	)
	s.Require().NoError(err)
}

func (s *StorageSuite) TestAddAndRemoveBannerFromSlot() {
	ctx := context.Background()
	db := s.store.DB()

	slotID := s.insertSlot(db, "slot1")
	bannerID := s.insertBanner(db, "banner1")

	err := s.store.AddBannerToSlot(ctx, slotID, bannerID)
	s.Require().NoError(err)

	err = s.store.AddBannerToSlot(ctx, slotID, bannerID)
	s.Require().NoError(err)

	err = s.store.RemoveBannerFromSlot(ctx, slotID, bannerID)
	s.Require().NoError(err)

	groupID := s.insertGroup(db, "group1")
	_, err = s.store.GetStats(ctx, slotID, groupID)
	s.Require().ErrorIs(err, domain.ErrNoBannersInSlot)
}

func (s *StorageSuite) TestRecordClickAndShow() {
	ctx := context.Background()
	db := s.store.DB()

	slotID := s.insertSlot(db, "slot1")
	bannerID := s.insertBanner(db, "banner1")
	groupID := s.insertGroup(db, "group1")

	s.Require().NoError(s.store.AddBannerToSlot(ctx, slotID, bannerID))

	s.Require().NoError(s.store.RecordShow(ctx, slotID, bannerID, groupID))
	s.Require().NoError(s.store.RecordClick(ctx, slotID, bannerID, groupID))

	var shows, clicks int64
	err := db.QueryRowContext(ctx,
		`SELECT shows, clicks FROM stats WHERE slot_id=$1 AND banner_id=$2 AND group_id=$3`,
		slotID, bannerID, groupID,
	).Scan(&shows, &clicks)
	s.Require().NoError(err)
	s.Require().Equal(int64(1), shows)
	s.Require().Equal(int64(1), clicks)
}

func (s *StorageSuite) TestGetStatsEmptySlot() {
	ctx := context.Background()
	db := s.store.DB()

	slotID := s.insertSlot(db, "empty_slot")
	groupID := s.insertGroup(db, "group1")

	_, err := s.store.GetStats(ctx, slotID, groupID)
	s.Require().ErrorIs(err, domain.ErrNoBannersInSlot)
}

func (s *StorageSuite) TestGetStatsReturnsAllBanners() {
	ctx := context.Background()
	db := s.store.DB()

	slotID := s.insertSlot(db, "slot1")
	groupID := s.insertGroup(db, "group1")

	var bannerIDs []int64
	for i := range 5 {
		id := s.insertBanner(db, fmt.Sprintf("banner%d", i))
		bannerIDs = append(bannerIDs, id)
		s.Require().NoError(s.store.AddBannerToSlot(ctx, slotID, id))
	}

	stats, err := s.store.GetStats(ctx, slotID, groupID)
	s.Require().NoError(err)
	s.Require().Len(stats, 5)

	seen := make(map[int64]bool)
	for _, st := range stats {
		seen[st.BannerID] = true
		s.Require().Equal(int64(0), st.Shows)
		s.Require().Equal(int64(0), st.Clicks)
	}
	for _, id := range bannerIDs {
		s.Require().True(seen[id], "banner %d missing from stats", id)
	}
}

func (s *StorageSuite) TestGetStatsAfterShowsAndClicks() {
	ctx := context.Background()
	db := s.store.DB()

	slotID := s.insertSlot(db, "slot1")
	groupID := s.insertGroup(db, "group1")
	bannerID := s.insertBanner(db, "banner1")

	s.Require().NoError(s.store.AddBannerToSlot(ctx, slotID, bannerID))
	s.Require().NoError(s.store.RecordShow(ctx, slotID, bannerID, groupID))
	s.Require().NoError(s.store.RecordShow(ctx, slotID, bannerID, groupID))
	s.Require().NoError(s.store.RecordClick(ctx, slotID, bannerID, groupID))

	stats, err := s.store.GetStats(ctx, slotID, groupID)
	s.Require().NoError(err)
	s.Require().Len(stats, 1)
	s.Require().Equal(int64(2), stats[0].Shows)
	s.Require().Equal(int64(1), stats[0].Clicks)
	s.Require().Equal(bannerID, stats[0].BannerID)
}

func (s *StorageSuite) insertSlot(db *sql.DB, desc string) int64 {
	var id int64
	err := db.QueryRowContext(context.Background(),
		`INSERT INTO slots (description) VALUES ($1) RETURNING id`, desc,
	).Scan(&id)
	s.Require().NoError(err)
	return id
}

func (s *StorageSuite) insertBanner(db *sql.DB, desc string) int64 {
	var id int64
	err := db.QueryRowContext(context.Background(),
		`INSERT INTO banners (description) VALUES ($1) RETURNING id`, desc,
	).Scan(&id)
	s.Require().NoError(err)
	return id
}

func (s *StorageSuite) insertGroup(db *sql.DB, desc string) int64 {
	var id int64
	err := db.QueryRowContext(context.Background(),
		`INSERT INTO social_groups (description) VALUES ($1) RETURNING id`, desc,
	).Scan(&id)
	s.Require().NoError(err)
	return id
}

func migrationsPath() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("cannot get caller info")
	}
	root := filepath.Join(filepath.Dir(filename), "..", "..")
	p := filepath.Join(root, "migrations", "00001_init.up.sql")
	if _, err := os.Stat(p); err != nil {
		panic(fmt.Sprintf("migration file not found: %s", p))
	}
	return p
}
