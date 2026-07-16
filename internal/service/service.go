package service

import (
	"context"
	"time"

	"github.com/jullss/banners-rotation/internal/broker/kafka"
	"github.com/jullss/banners-rotation/internal/domain"
	"github.com/jullss/banners-rotation/internal/storage"
)

type Service struct {
	storage  storage.Storage
	producer *kafka.Producer
}

func New(storage storage.Storage, producer *kafka.Producer) *Service {
	return &Service{storage: storage, producer: producer}
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
	_ = s.producer.Publish(ctx, kafka.Event{
		Type:     kafka.EventClick,
		SlotID:   slotID,
		BannerID: bannerID,
		GroupID:  groupID,
		Time:     time.Now(),
	})
	return nil
}

func (s *Service) ChooseBanner(ctx context.Context, slotID, groupID int64) (*domain.Banner, error) {
	banner, err := s.storage.ChooseBanner(ctx, slotID, groupID)
	if err != nil {
		return nil, err
	}
	_ = s.producer.Publish(ctx, kafka.Event{
		Type:     kafka.EventShow,
		SlotID:   slotID,
		BannerID: banner.ID,
		GroupID:  groupID,
		Time:     time.Now(),
	})
	return banner, nil
}
