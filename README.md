# pickup-rating

Rating system for [tf2pickup-org project](https://github.com/tf2pickup-org).

Components:
- **match-etl**: ETL for games data: loads games via API, calculates and saves rating changes.
- **pickup-ratings**: Website for rating data with leaderboards and game history for players.

### Run locally:
Requirements:
- Go 1.21
- https://github.com/casey/just

1. Install deps:
```bash
just install-deps
```
2. Start local db:
```bash
just local-up
```
3. Migrate local db:
```bash
just local-migrate up
```
4. Load games for pickup site with optional starting offset:
```bash
just match-etl --pickup-site tf2pickup.ru --starting-offset 2449
```
5. Start web service:
```bash
just start
```
