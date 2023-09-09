FROM busybox

RUN "/bin/busybox ls"

COPY bin/match-etl ./match-etl
COPY bin/pickup-ratings ./pickup-ratings

ENTRYPOINT ["./pickup-ratings"]