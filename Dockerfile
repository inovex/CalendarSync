FROM golang:1.23.0-bullseye AS Build

RUN mkdir /build
ADD . /build
WORKDIR /build
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates

RUN CGO_ENABLED=0 go build -o calendarsync -mod=vendor ./cmd/calendarsync

FROM scratch

VOLUME /etc/calendarsync
COPY --from=Build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=Build /build/calendarsync /app/
WORKDIR /app

ENTRYPOINT ["./calendarsync", "--config", "/etc/calendarsync/sync.yaml", "--port", "8085"]
