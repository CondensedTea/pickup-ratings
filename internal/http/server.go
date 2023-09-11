package http

import (
	"context"
	"embed"
	"fmt"
	"math"
	"net/http"

	"github.com/condensedtea/pickup-ratings/internal/db"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/template/html/v2"
)

const (
	defaultPickupSite  = "tf2pickup.ru"
	defaultPlayerClass = "scout"
)

//go:embed templates/*
var templateFS embed.FS

//go:embed assets/*
var assetFS embed.FS

type database interface {
	GetAvailablePickupSites(ctx context.Context) ([]string, error)
	GetLeaderboardForClass(ctx context.Context, playerClass, pickupSite string, offset, limit int) ([]db.LeaderboardEntry, error)
	GetPlayerRatingHistoryForClass(ctx context.Context, steamID int64, class string) ([]db.RatingUpdate, error)
	GetPlayerName(ctx context.Context, pickupSite string, steamID int64) (string, error)
}

type Server struct {
	db database

	app *fiber.App
}

func NewServer(db database) *Server {
	app := fiber.New(fiber.Config{

		AppName: "pickup-ratings",
		Views:   html.NewFileSystem(http.FS(templateFS), ".tmpl"),
	})

	s := &Server{app: app, db: db}

	s.app.Use("/assets", filesystem.New(filesystem.Config{
		MaxAge:     3600,
		Root:       http.FS(assetFS),
		PathPrefix: "assets",
	}))

	s.app.Get("/:pickupSite?", s.leaderboardsPage)
	s.app.Get("/:pickupSite/player/:steamID", s.playerPage)

	return s
}

func (s *Server) Run(port string) error {
	return s.app.Listen(":" + port)
}

func ratingLabel(v float64) string {
	return fmt.Sprintf("%.0f", math.Round(v*100))
}
