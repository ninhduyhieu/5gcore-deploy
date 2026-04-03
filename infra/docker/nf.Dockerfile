FROM debian:bookworm-slim

ARG B5GC_MODULE

WORKDIR /app

COPY helm/etrib5gc-main/bin/${B5GC_MODULE} /app/cmd
COPY helm/etrib5gc-main/config/${B5GC_MODULE}.json /app/config/${B5GC_MODULE}.json

RUN chmod +x /app/cmd

CMD ["/app/cmd", "-c", "/app/config/${B5GC_MODULE}.json"]
