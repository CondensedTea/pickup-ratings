FROM golang:1.21-bullseye

COPY bin/match-etl ./match-etl
COPY bin/pickup-ratings ./pickup-ratings
