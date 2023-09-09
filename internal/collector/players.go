package collector

import (
	"github.com/condensedtea/pickup-ratings/internal/db"
	"github.com/condensedtea/pickup-ratings/internal/tf2pickup"
)

type player struct {
	name      string
	avatarURL string
	steamID   int64
	team      string
	class     string
}

type playerSet struct {
	steamIDMapping map[int64]player
	steamIDs       []int64
}

func newPlayerSet(slots []tf2pickup.Slot) playerSet {
	steamIDMapping := make(map[int64]player, len(slots))
	steamIDs := make([]int64, len(slots))

	for i, slot := range slots {
		steamID := slot.Player.SteamId
		steamIDs[i] = steamID
		steamIDMapping[steamID] = player{
			name:      slot.Player.Name,
			avatarURL: slot.Player.Avatar.Small,
			steamID:   slot.Player.SteamId,
			team:      slot.Team,
			class:     slot.GameClass,
		}
	}

	return playerSet{
		steamIDMapping: steamIDMapping,
		steamIDs:       steamIDs,
	}
}

// filterRatingsByClass accepts slice of any ratings with given steamIDs and filters them based on playerSet player's classes
func (ps playerSet) filterRatingsByClass(ratings []db.PlayerRating) []db.PlayerRating {
	var playerRatings = make([]db.PlayerRating, len(ps.steamIDs))
	for _, steamIDRating := range ratings {
		p := ps.steamIDMapping[steamIDRating.SteamID]

		if p.class != steamIDRating.Class {
			continue
		}

		steamIDRating.Team = p.team

		playerRatings = append(playerRatings, steamIDRating)
	}

	return playerRatings
}

func (ps playerSet) bySteamID(id int64) player {
	return ps.steamIDMapping[id]
}

func defaultRating(p player) db.PlayerRating {
	const ratingValue = 16.0

	return db.PlayerRating{
		SteamID:          p.steamID,
		Class:            p.class,
		Rating:           ratingValue,
		UncertaintyValue: ratingValue / 3.0,
	}
}
