FROM busybox

RUN "ls -la"

COPY bin/match-etl ./match-etl
COPY bin/pickup-ratings ./pickup-ratings

ENTRYPOINT ["./pickup-ratings"]