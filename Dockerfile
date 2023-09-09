FROM scratch

COPY /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

COPY bin/match-etl ./match-etl
COPY bin/pickup-ratings ./pickup-ratings
