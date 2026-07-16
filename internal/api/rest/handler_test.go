package rest

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jullss/banners-rotation/internal/domain"
	"github.com/jullss/banners-rotation/internal/service"
)

func TestChooseBannerReturnsNotFoundForEmptySlot(t *testing.T) {
	svc := service.New(emptySlotStorage{}, nil, nil)
	mux := http.NewServeMux()
	NewHandler(svc).Register(mux)

	request := httptest.NewRequest(http.MethodGet, "/slots/1/choose?group_id=2", nil)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	require.Equal(t, http.StatusNotFound, response.Code)
}

type emptySlotStorage struct{}

func (emptySlotStorage) AddBannerToSlot(context.Context, int64, int64) error {
	return nil
}

func (emptySlotStorage) RemoveBannerFromSlot(context.Context, int64, int64) error {
	return nil
}

func (emptySlotStorage) GetBannersBySlot(context.Context, int64) ([]domain.Banner, error) {
	return nil, fmt.Errorf("get banners: %w", domain.ErrNoBannersInSlot)
}

func (emptySlotStorage) GetStats(context.Context, int64, int64) ([]domain.Stat, error) {
	return nil, nil
}

func (emptySlotStorage) RecordShow(context.Context, int64, int64, int64) error {
	return nil
}

func (emptySlotStorage) RecordClick(context.Context, int64, int64, int64) error {
	return nil
}
