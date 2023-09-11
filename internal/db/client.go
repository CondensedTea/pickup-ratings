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
	Name       string
	AvatarURL  string
	SteamID    int64
	PickupSite string
}

type Game struct {
	ID         int64
	Map        string
	PickupSite string
	BluScore   int64
	RedScore   int64
	Ts         string
	PickupID   string
}

type PlayerRating struct {
	ID      int64
	SteamID int64

	Rating           float64
	UncertaintyValue float64
	Result           string
	GamesPlayed      int64
	GamesTied        int64
	GamesWon         int64

	Class string
	Team  string
}

type RatingUpdate struct {
	GameID   int64
	PickupID string
	GameMap  string
	Rating   float64
	Result   string
	RedScore int64
	BluScore int64
	Date     string
	Time     string
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
	const query = `select game_id from game_history where pickup_site = $1 order by game_id desc limit 1`

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

func (c *Client) CreatePlayersBatch(ctx context.Context, players []Player, pickupSite string) error {
	const query = `insert into players(name, avatar_url, steam_id, pickup_site) values ($1, $2, $3, $4)`

	b := &pgx.Batch{}

	for _, player := range players {
		b.Queue(query, player.Name, player.AvatarURL, player.SteamID, pickupSite)
	}

	br := c.pool.SendBatch(ctx, b)
	defer br.Close()

	for i := 0; i < b.Len(); i++ {
		_, err := br.Exec()
		if err != nil {
			return fmt.Errorf("CreatePlayersBatch: %d: %w", i, err)
		}
	}

	return nil
}

func (c *Client) SaveGame(ctx context.Context, game Game) error {
	const query = `insert into game_history(game_id, game_map, pickup_site, red_score, blu_score, ts, pickup_id)
					values ($1, $2, $3, $4, $5, $6, $7)`

	_, err := c.pool.Exec(ctx, query, game.ID, game.Map, game.PickupSite, game.RedScore, game.BluScore, game.Ts, game.PickupID)
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
		    games_played,
			games_tied,
			games_won
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
		GamesTied        int64
		GamesWon         int64
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
			GamesTied:        r.GamesTied,
			GamesWon:         r.GamesWon,
		}
	}), nil
}

func (c *Client) LogRatingUpdates(ctx context.Context, gameID int64, pickupSite string, ratings []PlayerRating, ts string) error {
	const query = `insert into player_rating_history(game_id, pickup_site, leaderboard_id, rating_value, result, ts) values ($1, $2, $3, $4, $5, $6)`

	var b = &pgx.Batch{}

	for _, r := range ratings {
		b.Queue(query, gameID, pickupSite, r.ID, r.Rating, r.Result, ts)
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
	const query = `update player_leaderboard set
                              rating = $1,
                              uncertainty_value = $2,
                              games_played = $3,
                              games_tied = $4,
                              games_won = $5
                          where id = $6`

	var b = &pgx.Batch{}

	for _, r := range ratings {
		b.Queue(query, r.Rating, r.UncertaintyValue, r.GamesPlayed, r.GamesTied, r.GamesWon, r.ID)
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

type LeaderboardEntry struct {
	Name        string
	AvatarURL   string
	SteamID     int64
	Rating      float64
	GamesWon    int64
	GamesTied   int64
	GamesPlayed int64
}

func (c *Client) GetLeaderboardForClass(ctx context.Context, playerClass, pickupSite string, offset, limit int) ([]LeaderboardEntry, error) {
	const minPlayedGames = 15

	const query = `
		select
    		p.name,
    		p.avatar_url,
    		p.steam_id,
    		l.rating,
    		l.games_won,
    		l.games_tied,
    		l.games_played
		from player_leaderboard l
		join players p on l.player_steam_id = p.steam_id and l.pickup_site = p.pickup_site
		where l.pickup_site = $1
			and l.player_class = $2
			and l.games_played > $3
		order by rating desc
		offset $4 limit $5`

	rows, err := c.pool.Query(ctx, query, pickupSite, playerClass, minPlayedGames, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("GetLeaderboardForClass: failed to query leaderboard entries: %w", err)
	}

	results, err := pgx.CollectRows(rows, pgx.RowToStructByPos[LeaderboardEntry])
	if err != nil {
		return nil, fmt.Errorf("GetLeaderboardForClass: failed to parse rows: %w", err)
	}

	return results, nil
}

func (c *Client) GetAvailablePickupSites(ctx context.Context) ([]string, error) {
	const query = `select distinct pickup_site from game_history`

	rows, err := c.pool.Query(ctx, query)
	if err != nil {
		return nil, nil
	}

	return pgx.CollectRows(rows, pgx.RowTo[string])
}

func (c *Client) GetPlayerRatingHistoryForClass(ctx context.Context, steamID int64, class string) ([]RatingUpdate, error) {
	const query = `
		select
			gh.game_id,
			gh.pickup_id,
			gh.game_map,
			rating_value,
			result,
			gh.red_score,
			gh.blu_score,
			to_char(rh.ts, 'YYYY/MM/DD'),
			to_char(rh.ts, 'HH24:MM:SS')
		from player_rating_history rh
		join player_leaderboard pl on rh.leaderboard_id = pl.id
		join game_history gh on rh.game_id = gh.game_id
		where
			player_steam_id = $1 and player_class = $2
		order by rh.ts`

	rows, err := c.pool.Query(ctx, query, steamID, class)
	if err != nil {
		return nil, fmt.Errorf("GetPlayerRatingHistoryForClass: quering rows: %w", err)
	}

	return pgx.CollectRows(rows, pgx.RowToStructByPos[RatingUpdate])
}

func (c *Client) GetPlayerName(ctx context.Context, pickupSite string, steamID int64) (string, error) {
	const query = `select name from players where pickup_site = $1 and steam_id = $2`

	var playerName string
	if err := c.pool.QueryRow(ctx, query, pickupSite, steamID).Scan(&playerName); err != nil {
		return "", fmt.Errorf("GetPlayerName: quering rows: %w", err)
	}

	return playerName, nil
}
