package storage

import (
	"context"

	"github.com/jullss/banners-rotation/internal/domain"
)

type Storage interface {
	AddBannerToSlot(ctx context.Context, slotID, bannerID int64) error
	RemoveBannerFromSlot(ctx context.Context, slotID, bannerID int64) error
	RecordClick(ctx context.Context, slotID, bannerID, groupID int64) error
	ChooseBanner(ctx context.Context, slotID, groupID int64) (*domain.Banner, error)
}
