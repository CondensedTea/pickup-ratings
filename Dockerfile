FROM golang:1.21-bullseye as builder

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY bin/match-etl ./match-etl
COPY bin/pickup-ratings ./pickup-ratings
