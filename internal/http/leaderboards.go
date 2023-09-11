package http

import (
	"fmt"

	"github.com/condensedtea/pickup-ratings/internal/db"
	"github.com/gofiber/fiber/v2"
	"github.com/samber/lo"
)

type winrate struct {
	Wins   int
	Ties   int
	Losses int
}

type rating struct {
	Position  int
	AvatarURL string
	Name      string
	SteamID   int64
	Rating    string
	Winrate   winrate
}

func (s *Server) leaderboardsPage(ctx *fiber.Ctx) error {
	pickupSite := ctx.Params("pickupSite", defaultPickupSite)
	gameClass := ctx.Query("class", defaultPlayerClass)

	availableSites, err := s.db.GetAvailablePickupSites(ctx.Context())
	if err != nil {
		return fmt.Errorf("failed to get availible pickup sites: %w", err)
	}

	leaderboardEntries, err := s.db.GetLeaderboardForClass(ctx.Context(), gameClass, pickupSite, 0, 50)
	if err != nil {
		return err
	}

	ratings := lo.Map(leaderboardEntries, func(e db.LeaderboardEntry, i int) rating {
		return rating{
			Position:  i + 1,
			AvatarURL: e.AvatarURL,
			Name:      e.Name,
			SteamID:   e.SteamID,
			Rating:    ratingLabel(e.Rating),
			Winrate: winrate{
				Wins:   int(e.GamesWon),
				Ties:   int(e.GamesTied),
				Losses: int(e.GamesPlayed - (e.GamesWon + e.GamesTied)),
			},
		}
	})

	return ctx.Render("templates/leaderboards", fiber.Map{
		"PageTitle":      "Leaderboards",
		"PickupSite":     pickupSite,
		"AvailableSites": availableSites,
		"Ratings":        ratings,
	})
}
