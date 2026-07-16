package app

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/jullss/banners-rotation/internal/api/rest"
	"github.com/jullss/banners-rotation/internal/bandit"
	"github.com/jullss/banners-rotation/internal/broker/kafka"
	"github.com/jullss/banners-rotation/internal/config"
	"github.com/jullss/banners-rotation/internal/service"
	"github.com/jullss/banners-rotation/internal/storage/postgres"
)

func Run(ctx context.Context) error {
	cfg, err := config.New()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	store, err := postgres.New(cfg.DB.DSN)
	if err != nil {
		return fmt.Errorf("storage: %w", err)
	}
	defer store.Close()

	producer := kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.Topic)
	defer producer.Close()

	svc := service.New(store, bandit.UCB1{}, producer)

	mux := http.NewServeMux()
	rest.NewHandler(svc).Register(mux)

	srv := &http.Server{
		Addr:    cfg.HTTP.Addr,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		if err := srv.Shutdown(context.Background()); err != nil {
			log.Printf("http shutdown: %v", err)
		}
	}()

	log.Printf("starting http server on %s", cfg.HTTP.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http server: %w", err)
	}
	return nil
}
