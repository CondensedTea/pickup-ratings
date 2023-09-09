package http

import (
	"fmt"
	"math"

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
			Rating:    fmt.Sprintf("%.0f", math.Round(e.Rating*100)),
			Winrate: winrate{
				Wins:   int(e.GamesWon),
				Ties:   int(e.GamesTied),
				Losses: int(e.GamesPlayed - (e.GamesWon + e.GamesTied)),
			},
		}
	})

	return ctx.Render("templates/leaderboard", fiber.Map{
		"PickupSite": pickupSite,
		"Ratings":    ratings,
	})
}
