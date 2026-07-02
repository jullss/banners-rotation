package app

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/jullss/banners-rotation/internal/config"
)

func Run(ctx context.Context) error {
	cfg, err := config.New()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	srv := &http.Server{
		Addr: cfg.HTTP.Addr,
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
