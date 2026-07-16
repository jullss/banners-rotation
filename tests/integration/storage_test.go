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
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/testcontainers/testcontainers-go"

	"github.com/jullss/banners-rotation/internal/bandit"
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

	store, err := postgres.New(dsn, bandit.UCB1{})
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
	_, err = s.store.ChooseBanner(ctx, slotID, groupID)
	s.Require().Error(err)
}

func (s *StorageSuite) TestRecordClick() {
	ctx := context.Background()
	db := s.store.DB()

	slotID := s.insertSlot(db, "slot1")
	bannerID := s.insertBanner(db, "banner1")
	groupID := s.insertGroup(db, "group1")

	s.Require().NoError(s.store.AddBannerToSlot(ctx, slotID, bannerID))

	_, err := s.store.ChooseBanner(ctx, slotID, groupID)
	s.Require().NoError(err)

	err = s.store.RecordClick(ctx, slotID, bannerID, groupID)
	s.Require().NoError(err)

	var shows, clicks int64
	err = db.QueryRowContext(ctx,
		`SELECT shows, clicks FROM stats WHERE slot_id=$1 AND banner_id=$2 AND group_id=$3`,
		slotID, bannerID, groupID,
	).Scan(&shows, &clicks)
	s.Require().NoError(err)
	s.Require().Equal(int64(1), shows)
	s.Require().Equal(int64(1), clicks)
}

func (s *StorageSuite) TestChooseBannerEmptySlot() {
	ctx := context.Background()
	db := s.store.DB()

	slotID := s.insertSlot(db, "empty_slot")
	groupID := s.insertGroup(db, "group1")

	_, err := s.store.ChooseBanner(ctx, slotID, groupID)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "no banners in slot")
}

func (s *StorageSuite) TestAllBannersShownAtLeastOnce() {
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

	for range 50 {
		_, err := s.store.ChooseBanner(ctx, slotID, groupID)
		s.Require().NoError(err)
	}

	for _, bannerID := range bannerIDs {
		var shows int64
		err := db.QueryRowContext(ctx,
			`SELECT COALESCE(shows, 0) FROM stats WHERE slot_id=$1 AND banner_id=$2 AND group_id=$3`,
			slotID, bannerID, groupID,
		).Scan(&shows)
		s.Require().NoError(err)
		s.Require().Greater(shows, int64(0), "banner %d was never shown", bannerID)
	}
}

func (s *StorageSuite) TestPopularBannerShownMore() {
	ctx := context.Background()
	db := s.store.DB()

	slotID := s.insertSlot(db, "slot1")
	groupID := s.insertGroup(db, "group1")

	banner1ID := s.insertBanner(db, "banner1")
	banner2ID := s.insertBanner(db, "banner2")
	banner3ID := s.insertBanner(db, "banner3")

	s.Require().NoError(s.store.AddBannerToSlot(ctx, slotID, banner1ID))
	s.Require().NoError(s.store.AddBannerToSlot(ctx, slotID, banner2ID))
	s.Require().NoError(s.store.AddBannerToSlot(ctx, slotID, banner3ID))

	for range 300 {
		banner, err := s.store.ChooseBanner(ctx, slotID, groupID)
		s.Require().NoError(err)
		if banner.ID == banner2ID {
			s.Require().NoError(s.store.RecordClick(ctx, slotID, banner2ID, groupID))
		}
	}

	var shows1, shows2, shows3 int64
	_ = db.QueryRowContext(ctx,
		`SELECT COALESCE(shows,0) FROM stats WHERE slot_id=$1 AND banner_id=$2 AND group_id=$3`,
		slotID, banner1ID, groupID,
	).Scan(&shows1)
	_ = db.QueryRowContext(ctx,
		`SELECT COALESCE(shows,0) FROM stats WHERE slot_id=$1 AND banner_id=$2 AND group_id=$3`,
		slotID, banner2ID, groupID,
	).Scan(&shows2)
	_ = db.QueryRowContext(ctx,
		`SELECT COALESCE(shows,0) FROM stats WHERE slot_id=$1 AND banner_id=$2 AND group_id=$3`,
		slotID, banner3ID, groupID,
	).Scan(&shows3)

	s.Require().Greater(shows2, shows1, "popular banner2 should have more shows than banner1")
	s.Require().Greater(shows2, shows3, "popular banner2 should have more shows than banner3")
}

type fixedChooser struct{ id int64 }

func (f fixedChooser) Choose(_ []domain.Stat) int64 { return f.id }

func (s *StorageSuite) TestChooseBannerWithFixedChooser() {
	ctx := context.Background()
	db := s.store.DB()

	slotID := s.insertSlot(db, "slot1")
	groupID := s.insertGroup(db, "group1")
	banner1ID := s.insertBanner(db, "banner1")
	banner2ID := s.insertBanner(db, "banner2")

	dsn, err := s.container.ConnectionString(ctx, "sslmode=disable")
	s.Require().NoError(err)

	fixedStore, err := postgres.New(dsn, fixedChooser{id: banner1ID})
	s.Require().NoError(err)
	defer fixedStore.Close()

	s.Require().NoError(fixedStore.AddBannerToSlot(ctx, slotID, banner1ID))
	s.Require().NoError(fixedStore.AddBannerToSlot(ctx, slotID, banner2ID))

	for range 5 {
		chosen, err := fixedStore.ChooseBanner(ctx, slotID, groupID)
		s.Require().NoError(err)
		s.Require().Equal(banner1ID, chosen.ID)
	}

	var shows1, shows2 int64
	_ = db.QueryRowContext(ctx,
		`SELECT COALESCE(shows,0) FROM stats WHERE slot_id=$1 AND banner_id=$2 AND group_id=$3`,
		slotID, banner1ID, groupID,
	).Scan(&shows1)
	_ = db.QueryRowContext(ctx,
		`SELECT COALESCE(shows,0) FROM stats WHERE slot_id=$1 AND banner_id=$2 AND group_id=$3`,
		slotID, banner2ID, groupID,
	).Scan(&shows2)

	s.Require().Equal(int64(5), shows1)
	s.Require().Equal(int64(0), shows2)
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
