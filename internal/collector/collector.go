package collector

import (
	"context"

	"github.com/condensedtea/pickup-ratings/internal/db"
	"github.com/condensedtea/pickup-ratings/internal/tf2pickup"
	"github.com/eullerpereira94/openskill"
	"github.com/samber/lo"
	"golang.org/x/exp/slog"
)

type database interface {
	GetLastGameID(ctx context.Context, pickupSite string) (int, error)
	GetUnknownSteamIDs(ctx context.Context, steamIDs []int64, pickupSite string) ([]int64, error)
	CreatePlayersBatch(ctx context.Context, players []db.Player, pickupSite string) error
	SaveGame(ctx context.Context, game db.Game) error
	CreatePlayerRatings(ctx context.Context, ratings []db.PlayerRating, pickupSite string) error
	GetPlayerRatingsForSteamIDs(ctx context.Context, steamIDs []int64, pickupSite string) ([]db.PlayerRating, error)
	LogRatingUpdates(ctx context.Context, gameID int64, pickupSite string, ratings []db.PlayerRating) error
	UpdatePlayerRatings(ctx context.Context, ratings []db.PlayerRating) error
}

type pickupAPI interface {
	LoadNewGames(ctx context.Context, offset int) ([]tf2pickup.Result, error)
}

type Collector struct {
	pickupSite string

	db  database
	api pickupAPI
}

func New(db database, api pickupAPI, pickupSite string) *Collector {
	return &Collector{db: db, api: api, pickupSite: pickupSite}
}

func (c *Collector) CollectGames(ctx context.Context) error {
	offset, err := c.db.GetLastGameID(ctx, c.pickupSite)
	if err != nil {
		return err
	}

	games, err := c.api.LoadNewGames(ctx, offset)
	if err != nil {
		return err
	}

	for _, game := range games {
		slog.Info("processing game", "number", game.Number)
		if err = c.processGame(ctx, game); err != nil {
			return err
		}
	}

	return nil
}

func (c *Collector) processGame(ctx context.Context, game tf2pickup.Result) (err error) {
	// handle ongoing games
	if game.State == "in progress" {
		return nil
	}

	dbGame := db.Game{
		ID:         game.Number,
		PickupSite: c.pickupSite,
		RedScore:   game.Score.Red,
		BluScore:   game.Score.Blu,
		// todo: use game.EndedAt as timestamp in database
	}

	if err = c.db.SaveGame(ctx, dbGame); err != nil {
		return err
	}

	// handle broken games
	if game.State != "ended" {
		slog.Info("ignored game with broken state", "state", game.State, "game_number", game.Number)
		return nil
	}

	players := newPlayerSet(game.Slots)

	newSteamIDs, err := c.createNewPlayers(ctx, players)
	if err != nil {
		return err
	}

	newRatings := lo.Map(newSteamIDs, func(steamID int64, _ int) db.PlayerRating {
		return defaultRating(players.bySteamID(steamID))
	})

	if err = c.db.CreatePlayerRatings(ctx, newRatings, c.pickupSite); err != nil {
		return err
	}

	slog.Debug("new players created")

	// calculate ratings diffs
	steamIDRatings, err := c.db.GetPlayerRatingsForSteamIDs(ctx, players.steamIDs, c.pickupSite)
	if err != nil {
		return err
	}

	playerRatings := players.filterRatingsByClass(steamIDRatings)

	teamRatings := lo.GroupBy(playerRatings, func(pr db.PlayerRating) string {
		return pr.Team
	})

	redRating, bluRating := teamRatings["red"], teamRatings["blu"]

	newBluRating, newRedRating := rateTeams(redRating, bluRating, game.Score.Red, game.Score.Blu)

	ratings := append(newRedRating, newBluRating...)

	slog.Debug("new ratings calculated")

	if err = c.db.LogRatingUpdates(ctx, game.Number, c.pickupSite, ratings); err != nil {
		return err
	}

	slog.Debug("ratings logged")

	if err = c.db.UpdatePlayerRatings(ctx, ratings); err != nil {
		return err
	}

	slog.Debug("ratings updated")

	return nil
}

func rateTeams(t1, t2 []db.PlayerRating, redScore, bluScore int64) ([]db.PlayerRating, []db.PlayerRating) {
	t1Ratings := playerRatingsToOpenSkillTeam(t1)
	t2Ratings := playerRatingsToOpenSkillTeam(t2)

	teams := openskill.Rate([]openskill.Team{t1Ratings, t2Ratings}, openskill.Options{Scores: []int64{redScore, bluScore}})

	t1Ratings, t2Ratings = teams[0], teams[1]

	redResult, bluResult := gameResults(redScore, bluScore)

	t1 = updateTeamRatings(t1, t1Ratings, redResult)
	t2 = updateTeamRatings(t2, t2Ratings, bluResult)

	return t1, t2
}

func updateTeamRatings(team []db.PlayerRating, ratings openskill.Team, result string) []db.PlayerRating {
	for i, p := range team {
		playerRatings := ratings[i]

		p.DiffValue = playerRatings.AveragePlayerSkill - p.Rating
		p.Rating = playerRatings.AveragePlayerSkill
		p.UncertaintyValue = playerRatings.SkillUncertaintyDegree
		p.Result = result
		p.GamesPlayed++

		switch result {
		case "win":
			p.GamesWon++
		case "tie":
			p.GamesTied++
		}

		team[i] = p
	}

	return team
}

func playerRatingsToOpenSkillTeam(team []db.PlayerRating) openskill.Team {
	var ratings = make([]*openskill.Rating, len(team))
	for i, p := range team {
		ratings[i] = openskill.NewRating(&openskill.NewRatingParams{
			AveragePlayerSkill:     p.Rating,
			SkillUncertaintyDegree: p.UncertaintyValue,
		}, nil)
	}

	return openskill.NewTeam(ratings...)
}

func (c *Collector) createNewPlayers(ctx context.Context, players playerSet) (newSteamIDs []int64, err error) {
	unknownSteamIDs, err := c.db.GetUnknownSteamIDs(ctx, players.steamIDs, c.pickupSite)
	if err != nil {
		return nil, err
	}

	var dbPlayers = make([]db.Player, len(unknownSteamIDs))
	for i, steamID := range unknownSteamIDs {
		p := players.bySteamID(steamID)
		dbPlayers[i] = db.Player{
			Name:       p.name,
			AvatarURL:  p.avatarURL,
			SteamID:    p.steamID,
			PickupSite: c.pickupSite,
		}
	}

	if err = c.db.CreatePlayersBatch(ctx, dbPlayers, c.pickupSite); err != nil {
		return nil, err
	}

	return unknownSteamIDs, nil
}

func gameResults(redScore, bluScore int64) (redResult, bluResult string) {
	if redScore == bluScore {
		return "tie", "tie"
	}

	if redScore > bluScore {
		return "win", "loss"
	}

	return "loss", "win"
}
