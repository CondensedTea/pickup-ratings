package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"log/slog"

	"github.com/condensedtea/pickup-ratings/internal/collector"
	"github.com/condensedtea/pickup-ratings/internal/db"
	"github.com/condensedtea/pickup-ratings/internal/tf2pickup"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	pickupSite, ok := os.LookupEnv("PICKUP_SITE")
	if !ok {
		log.Fatal("could not find PICKUP_SITE env")
	}

	dbDsn, ok := os.LookupEnv("DB_DSN")
	if !ok {
		log.Fatal("could not find DB_DSN env")
	}

	dbClient, err := db.NewClient(ctx, dbDsn)
	if err != nil {
		log.Fatalf("failed to init db client: %w", err)
	}

	pickupApi := tf2pickup.NewClient(pickupSite, 150, http.DefaultTransport)

	c := collector.New(dbClient, pickupApi, pickupSite)

	slog.Info("collecting games")

	if err = c.CollectGames(ctx); err != nil {
		slog.Error("failed to run collector", "error", err)
		os.Exit(1)
	}
}
