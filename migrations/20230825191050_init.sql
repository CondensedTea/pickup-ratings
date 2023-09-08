-- +goose Up
-- +goose StatementBegin
create table players (
    name text,
    avatar_url text,
    steam_id bigint not null,
    pickup_site text not null,

    primary key (steam_id, pickup_site)
);

create table player_leaderboard (
    id bigint primary key generated always as identity,
    pickup_site text not null,
    player_steam_id bigint not null,
    player_class text not null,

    rating float4 not null,
    uncertainty_value float4 not null,
    games_played bigint default 1
);

create table player_rating_history (
    id bigint primary key generated always as identity,
    game_id int not null,
	pickup_site text not null,
    leaderboard_id bigint not null,
    rating_diff float4 not null,
    result text not null,
    ts timestamp default now()
);

-- info about single game on given pickup site
create table game_history (
    game_id int not null,
	pickup_site text not null,
	blu_score int not null,
    red_score int not null,
	ts timestamp default now(),

	primary key (game_id, pickup_site)
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table game_history;
drop table player_rating_history;
drop table player_leaderboard;
drop table players;
-- +goose StatementEnd
