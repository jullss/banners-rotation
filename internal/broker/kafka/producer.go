package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

type EventType string

const (
	EventShow  EventType = "show"
	EventClick EventType = "click"
)

type Event struct {
	Type     EventType `json:"type"`
	SlotID   int64     `json:"slot_id"`
	BannerID int64     `json:"banner_id"`
	GroupID  int64     `json:"group_id"`
	Time     time.Time `json:"timestamp"`
}

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string, topic string) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Topic:    topic,
			Balancer: &kafka.LeastBytes{},
		},
	}
}

func (p *Producer) Publish(ctx context.Context, event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	if err := p.writer.WriteMessages(ctx, kafka.Message{Value: data}); err != nil {
		return fmt.Errorf("write kafka message: %w", err)
	}
	return nil
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
