package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	flag "github.com/spf13/pflag"

	"log/slog"

	"github.com/condensedtea/pickup-ratings/internal/collector"
	"github.com/condensedtea/pickup-ratings/internal/db"
	"github.com/condensedtea/pickup-ratings/internal/tf2pickup"
)

var (
	pickupSite    string
	gamesPageSize int
)

func main() {
	flag.StringVar(&pickupSite, "pickup-site", "", "Host of the pickup site to load games from")
	flag.IntVar(&gamesPageSize, "games-page-size", 200, "Amount of games per page for API requests")
	flag.Parse()

	if pickupSite == "" {
		log.Fatal("--pickup-site must be specified")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	dbDsn, ok := os.LookupEnv("DB_DSN")
	if !ok {
		log.Fatal("could not find DB_DSN env")
	}

	dbClient, err := db.NewClient(ctx, dbDsn)
	if err != nil {
		log.Fatalf("failed to init db client: %s", err)
	}

	pickupApi := tf2pickup.NewClient(pickupSite, gamesPageSize, http.DefaultTransport)

	c := collector.New(dbClient, pickupApi, pickupSite)

	slog.Info("collecting games")

	if err = c.CollectGames(ctx); err != nil {
		log.Fatalf("failed to run collector: %s", err)
	}
}
