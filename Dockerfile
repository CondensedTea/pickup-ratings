FROM debian:12

COPY /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY bin/match-etl ./match-etl
COPY bin/pickup-ratings ./pickup-ratings
