package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/lo"
)

var classes = []string{"medic", "demoman", "soldier", "scout"}

type Player struct {
	SteamID    int64
	PickupSite string
}

type Game struct {
	ID         int64
	PickupSite string

	BluScore int64

	RedScore int64
}

type PlayerRating struct {
	ID      int64
	SteamID int64

	Rating           float64
	UncertaintyValue float64
	DiffValue        float64 `db:"-"`
	Result           string
	GamesPlayed      int64

	Class string
	Team  string

	PickupSite string
}

type RatingUpdate struct {
	ID         int64
	GameID     int64
	PickupSite string

	PlayerSteamID int64
	PlayerClass   string
}

type Client struct {
	pool *pgxpool.Pool
}

func NewClient(ctx context.Context, dsn string) (*Client, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	return &Client{pool: pool}, nil
}

func (c *Client) GetLastGameID(ctx context.Context, pickupSite string) (int, error) {
	const query = `select game_id from game_history where pickup_site = $1 order by ts desc limit 1`

	var gameID int
	if err := c.pool.QueryRow(ctx, query, pickupSite).Scan(&gameID); err != nil {
		return 0, fmt.Errorf("GetLastGameID: %w", err)
	}

	return gameID, nil
}

func (c *Client) GetUnknownSteamIDs(ctx context.Context, steamIDs []int64, pickupSite string) ([]int64, error) {
	const query = `
		select steam_id from unnest($1::bigint[]) as steam_ids(steam_id)
		where not exists(select 1 from players p where p.steam_id = steam_ids.steam_id and pickup_site = $2)`

	rows, err := c.pool.Query(ctx, query, steamIDs, pickupSite)
	if err != nil {
		return nil, fmt.Errorf("GetUnknownSteamIDs: %w", err)
	}

	return pgx.CollectRows(rows, pgx.RowTo[int64])
}

func (c *Client) CreatePlayerRatings(ctx context.Context, ratings []PlayerRating, pickupSite string) error {
	const query = `
			insert into player_leaderboard(pickup_site, player_steam_id, player_class, rating, uncertainty_value)
			values ($1, $2, $3, $4, $5)`

	b := &pgx.Batch{}

	for _, r := range ratings {
		for _, class := range classes {
			b.Queue(query, pickupSite, r.SteamID, class, r.Rating, r.UncertaintyValue)
		}
	}

	br := c.pool.SendBatch(ctx, b)
	defer br.Close()

	for i := 0; i < b.Len(); i++ {
		_, err := br.Exec()
		if err != nil {
			return fmt.Errorf("CreatePlayerRatings: %d: %w", i, err)
		}
	}

	return nil
}

func (c *Client) CreatePlayersBatch(ctx context.Context, steamIDs []int64, pickupSite string) error {
	const query = `
		insert into players(steam_id, pickup_site) 
		select steam_id, $2 from unnest($1::bigint[]) as steam_ids(steam_id)`

	_, err := c.pool.Exec(ctx, query, steamIDs, pickupSite)
	if err != nil {
		return fmt.Errorf("CreatePlayersBatch: %w", err)
	}

	return nil
}

func (c *Client) SaveGame(ctx context.Context, game Game) error {
	const query = `insert into game_history(game_id, pickup_site, red_score, blu_score) values ($1, $2, $3, $4)`

	_, err := c.pool.Exec(ctx, query, game.ID, game.PickupSite, game.RedScore, game.BluScore)
	if err != nil {
		return fmt.Errorf("SaveGame: %w", err)
	}

	return nil
}

func (c *Client) GetPlayerRatingsForSteamIDs(ctx context.Context, steamIDs []int64, pickupSite string) ([]PlayerRating, error) {
	const query = `
		select
		    id,
		    player_steam_id,
		    rating,
		    uncertainty_value,
		    player_class,
		    games_played
		from player_leaderboard
		where pickup_site = $1 and player_steam_id = any($2::bigint[])`

	rows, err := c.pool.Query(ctx, query, pickupSite, steamIDs)
	if err != nil {
		return nil, fmt.Errorf("GetPlayerRatingsForSteamIDs: %w", err)
	}

	type result struct {
		ID               int64
		PlayerSteamID    int64
		Rating           float64
		UncertaintyValue float64
		PlayerClass      string
		GamesPlayed      int64
	}

	results, err := pgx.CollectRows(rows, pgx.RowToStructByPos[result])
	if err != nil {
		return nil, fmt.Errorf("collecting results for GetPlayerRatingsForSteamIDs: %w", err)
	}

	return lo.Map(results, func(r result, _ int) PlayerRating {
		return PlayerRating{
			ID:               r.ID,
			SteamID:          r.PlayerSteamID,
			Rating:           r.Rating,
			UncertaintyValue: r.UncertaintyValue,
			Class:            r.PlayerClass,
			GamesPlayed:      r.GamesPlayed,
		}
	}), nil
}

func (c *Client) LogRatingUpdates(ctx context.Context, gameID int64, pickupSite string, ratings []PlayerRating) error {
	const query = `insert into player_rating_history(game_id, pickup_site, leaderboard_id, rating_diff, result) values ($1, $2, $3, $4, $5)`

	var b = &pgx.Batch{}

	for _, r := range ratings {
		b.Queue(query, gameID, pickupSite, r.ID, r.DiffValue, r.Result)
	}

	br := c.pool.SendBatch(ctx, b)
	defer br.Close()

	for i := range ratings {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("LogRatingUpdates: %d: %w", i, err)
		}
	}

	return nil
}

func (c *Client) UpdatePlayerRatings(ctx context.Context, ratings []PlayerRating) error {
	const query = `update player_leaderboard set rating = $1, uncertainty_value = $2, games_played = $3 where id = $4`

	var b = &pgx.Batch{}

	for _, r := range ratings {
		b.Queue(query, r.Rating, r.UncertaintyValue, r.GamesPlayed, r.ID)
	}

	br := c.pool.SendBatch(ctx, b)
	defer br.Close()

	for i := range ratings {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("UpdatePlayerRatings: %d: %w", i, err)
		}
	}

	return nil
}
