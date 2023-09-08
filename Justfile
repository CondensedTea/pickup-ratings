set positional-arguments

LOCAL_BIN := invocation_directory() + "/bin"
LOCAL_DSN := "postgres://postgres:password@localhost:5432/pickup-ratings?sslmode=disable"

default:
  @just --list

install-deps:
    if [ ! -f "{{ LOCAL_BIN }}/goose" ] ; then GOBIN={{ LOCAL_BIN }} go install github.com/pressly/goose/v3/cmd/goose@latest; fi

_build-match-etl:
    go build -o bin/match-etl ./cmd/match-etl

_build-player-etl:
    go build -o bin/player-etl ./cmd/player-etl

# Build binaries
build: _build-match-etl
#_build-player-etl

test:
    echo {{ file_name("cmd/player-etl") }}

local-up:
    docker-compose up -d

local-down:
    docker-compose down

local-migrate *args='': install-deps
    ./bin/goose -dir=migrations/ postgres {{ LOCAL_DSN }} "$@"
