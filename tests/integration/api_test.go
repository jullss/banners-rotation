//go:build integration

package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/jullss/banners-rotation/internal/api/rest"
	"github.com/jullss/banners-rotation/internal/bandit"
	"github.com/jullss/banners-rotation/internal/broker/kafka"
	"github.com/jullss/banners-rotation/internal/domain"
	"github.com/jullss/banners-rotation/internal/service"
	"github.com/jullss/banners-rotation/internal/storage/postgres"
)

type noopPublisher struct{}

func (noopPublisher) Publish(_ context.Context, _ kafka.Event) error { return nil }

type APISuite struct {
	suite.Suite
	container *tcpostgres.PostgresContainer
	store     *postgres.Storage
	server    *httptest.Server
	client    *http.Client
	db        *sql.DB
}

func TestAPISuite(t *testing.T) {
	suite.Run(t, new(APISuite))
}

func (s *APISuite) SetupSuite() {
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
	s.db = store.DB()

	svc := service.New(store, noopPublisher{})
	mux := http.NewServeMux()
	rest.NewHandler(svc).Register(mux)
	s.server = httptest.NewServer(mux)
	s.client = s.server.Client()
}

func (s *APISuite) TearDownSuite() {
	if s.server != nil {
		s.server.Close()
	}
	if s.store != nil {
		s.store.Close()
	}
	if s.container != nil {
		s.container.Terminate(context.Background()) //nolint:errcheck
	}
}

func (s *APISuite) TearDownTest() {
	_, err := s.db.ExecContext(context.Background(),
		`TRUNCATE stats, slot_banners, banners, slots, social_groups RESTART IDENTITY CASCADE`,
	)
	s.Require().NoError(err)
}

func (s *APISuite) url(path string) string {
	return s.server.URL + path
}

func (s *APISuite) insertSlotDB(desc string) int64 {
	var id int64
	err := s.db.QueryRowContext(context.Background(),
		`INSERT INTO slots (description) VALUES ($1) RETURNING id`, desc,
	).Scan(&id)
	s.Require().NoError(err)
	return id
}

func (s *APISuite) insertBannerDB(desc string) int64 {
	var id int64
	err := s.db.QueryRowContext(context.Background(),
		`INSERT INTO banners (description) VALUES ($1) RETURNING id`, desc,
	).Scan(&id)
	s.Require().NoError(err)
	return id
}

func (s *APISuite) insertGroupDB(desc string) int64 {
	var id int64
	err := s.db.QueryRowContext(context.Background(),
		`INSERT INTO social_groups (description) VALUES ($1) RETURNING id`, desc,
	).Scan(&id)
	s.Require().NoError(err)
	return id
}

func (s *APISuite) addBannerToSlot(slotID, bannerID int64) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost,
		s.url(fmt.Sprintf("/slots/%d/banners/%d", slotID, bannerID)), nil)
	s.Require().NoError(err)
	resp, err := s.client.Do(req)
	s.Require().NoError(err)
	resp.Body.Close()
	s.Require().Equal(http.StatusNoContent, resp.StatusCode)
}

func (s *APISuite) chooseBanner(slotID, groupID int64) *domain.Banner {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet,
		s.url(fmt.Sprintf("/slots/%d/choose?group_id=%d", slotID, groupID)), nil)
	s.Require().NoError(err)
	resp, err := s.client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Require().Equal(http.StatusOK, resp.StatusCode)
	var b domain.Banner
	s.Require().NoError(json.NewDecoder(resp.Body).Decode(&b))
	return &b
}

func (s *APISuite) recordClick(slotID, bannerID, groupID int64) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost,
		s.url(fmt.Sprintf("/slots/%d/banners/%d/click?group_id=%d", slotID, bannerID, groupID)), nil)
	s.Require().NoError(err)
	resp, err := s.client.Do(req)
	s.Require().NoError(err)
	resp.Body.Close()
	s.Require().Equal(http.StatusNoContent, resp.StatusCode)
}

func (s *APISuite) TestAPI_AllBannersShownAtLeastOnce() {
	slotID := s.insertSlotDB("slot1")
	groupID := s.insertGroupDB("group1")

	var bannerIDs []int64
	for i := range 5 {
		id := s.insertBannerDB(fmt.Sprintf("banner%d", i))
		bannerIDs = append(bannerIDs, id)
		s.addBannerToSlot(slotID, id)
	}

	seen := make(map[int64]bool)
	for range 50 {
		b := s.chooseBanner(slotID, groupID)
		seen[b.ID] = true
	}

	for _, id := range bannerIDs {
		s.Require().True(seen[id], "banner %d was never shown", id)
	}
}

func (s *APISuite) TestAPI_PopularBannerShownMore() {
	slotID := s.insertSlotDB("slot1")
	groupID := s.insertGroupDB("group1")

	banner1ID := s.insertBannerDB("banner1")
	banner2ID := s.insertBannerDB("banner2")
	banner3ID := s.insertBannerDB("banner3")

	s.addBannerToSlot(slotID, banner1ID)
	s.addBannerToSlot(slotID, banner2ID)
	s.addBannerToSlot(slotID, banner3ID)

	counts := make(map[int64]int)
	for range 300 {
		b := s.chooseBanner(slotID, groupID)
		counts[b.ID]++
		if b.ID == banner2ID {
			s.recordClick(slotID, banner2ID, groupID)
		}
	}

	s.Require().Greater(counts[banner2ID], counts[banner1ID],
		"popular banner2 should be shown more than banner1")
	s.Require().Greater(counts[banner2ID], counts[banner3ID],
		"popular banner2 should be shown more than banner3")
}
