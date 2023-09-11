package main

import (
	"context"
	"log"
	"os"

	"github.com/condensedtea/pickup-ratings/internal/db"
	"github.com/condensedtea/pickup-ratings/internal/http"
)

func main() {
	ctx := context.Background()

	dbClient, err := db.NewClient(ctx, os.Getenv("DB_DSN"))
	if err != nil {
		log.Fatal(err)
	}

	server := http.NewServer(dbClient)

	if err = server.Run(os.Getenv("PORT")); err != nil {
		log.Fatalf("failed to run server: %s", err)
	}
}
