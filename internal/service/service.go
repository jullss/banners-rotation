package service

import (
	"context"
	"fmt"
	"time"

	"github.com/jullss/banners-rotation/internal/bandit"
	"github.com/jullss/banners-rotation/internal/broker/kafka"
	"github.com/jullss/banners-rotation/internal/domain"
	"github.com/jullss/banners-rotation/internal/storage"
)

type Publisher interface {
	Publish(ctx context.Context, event kafka.Event) error
}

type Service struct {
	storage   storage.Storage
	chooser   bandit.Chooser
	publisher Publisher
}

func New(storage storage.Storage, chooser bandit.Chooser, publisher Publisher) *Service {
	return &Service{storage: storage, chooser: chooser, publisher: publisher}
}

func (s *Service) AddBannerToSlot(ctx context.Context, slotID, bannerID int64) error {
	return s.storage.AddBannerToSlot(ctx, slotID, bannerID)
}

func (s *Service) RemoveBannerFromSlot(ctx context.Context, slotID, bannerID int64) error {
	return s.storage.RemoveBannerFromSlot(ctx, slotID, bannerID)
}

func (s *Service) RecordClick(ctx context.Context, slotID, bannerID, groupID int64) error {
	if err := s.storage.RecordClick(ctx, slotID, bannerID, groupID); err != nil {
		return err
	}
	_ = s.publisher.Publish(ctx, kafka.Event{
		Type:     kafka.EventClick,
		SlotID:   slotID,
		BannerID: bannerID,
		GroupID:  groupID,
		Time:     time.Now(),
	})
	return nil
}

func (s *Service) ChooseBanner(ctx context.Context, slotID, groupID int64) (*domain.Banner, error) {
	banners, err := s.storage.GetBannersBySlot(ctx, slotID)
	if err != nil {
		return nil, err
	}

	stats, err := s.storage.GetStats(ctx, slotID, groupID)
	if err != nil {
		return nil, err
	}

	bannerID := s.chooser.Choose(stats)

	var banner *domain.Banner
	for i := range banners {
		if banners[i].ID == bannerID {
			banner = &banners[i]
			break
		}
	}
	if banner == nil {
		return nil, fmt.Errorf("chosen banner %d not found in slot", bannerID)
	}

	if err := s.storage.RecordShow(ctx, slotID, bannerID, groupID); err != nil {
		return nil, err
	}

	_ = s.publisher.Publish(ctx, kafka.Event{
		Type:     kafka.EventShow,
		SlotID:   slotID,
		BannerID: bannerID,
		GroupID:  groupID,
		Time:     time.Now(),
	})
	return banner, nil
}
