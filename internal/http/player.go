package http

import (
	"fmt"

	"github.com/condensedtea/pickup-ratings/internal/db"
	"github.com/gofiber/fiber/v2"
	"github.com/samber/lo"
)

type playerRatingEntry struct {
	GameID      int
	PickupID    string
	Map         string
	Rating      string
	Result      string
	RatingClass string
	RatingDiff  string
	RedScore    int
	BluScore    int
	Date        string
	Time        string
}

func (s *Server) playerPage(ctx *fiber.Ctx) error {
	pickupSite := ctx.Params("pickupSite")
	gameClass := ctx.Query("class", defaultPlayerClass)

	steamID, err := ctx.ParamsInt("steamID")
	if err != nil {
		return &fiber.Error{
			Code:    fiber.StatusBadRequest,
			Message: fmt.Sprintf("failed to parse steamID: %s", err),
		}
	}

	availableSites, err := s.db.GetAvailablePickupSites(ctx.Context())
	if err != nil {
		return fmt.Errorf("failed to get availible pickup sites: %w", err)
	}

	history, err := s.db.GetPlayerRatingHistoryForClass(ctx.Context(), int64(steamID), gameClass)
	if err != nil {
		return fmt.Errorf("failed to get player's history: %s", err)
	}

	playerName, err := s.db.GetPlayerName(ctx.Context(), pickupSite, int64(steamID))
	if err != nil {
		return fmt.Errorf("failed to get player's name: %w", err)
	}

	var lastRatingValue float64
	entries := lo.Map(history, func(u db.RatingUpdate, _ int) playerRatingEntry {
		e := playerRatingEntry{
			GameID:     int(u.GameID),
			PickupID:   u.PickupID,
			Map:        u.GameMap,
			Rating:     ratingLabel(u.Rating),
			RatingDiff: ratingDiffLabel(u.Rating, lastRatingValue),
			Result:     u.Result,
			RedScore:   int(u.RedScore),
			BluScore:   int(u.BluScore),
			Date:       u.Date,
			Time:       u.Time,
		}

		lastRatingValue = u.Rating
		return e
	})

	return ctx.Render("templates/player", fiber.Map{
		"PageTitle":      playerName,
		"PickupSite":     pickupSite,
		"AvailableSites": availableSites,
		"RatingEntries":  lo.Reverse(entries),
		"SteamID":        steamID,
	})
}

func ratingDiffLabel(rating float64, lastValue float64) string {
	if lastValue == 0.0 {
		return ratingLabel(0)
	} else {
		ratingDiff := rating - lastValue
		switch {
		case ratingDiff > 0:
			return "+" + ratingLabel(ratingDiff)
		case ratingDiff < 0:
			return ratingLabel(ratingDiff)
		default:
			return ratingLabel(ratingDiff)
		}
	}
}
