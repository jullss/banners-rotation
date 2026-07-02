package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jullss/banners-rotation/internal/app"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := app.Run(ctx); err != nil {
		log.Fatalf("application error: %v", err)
	}
}
