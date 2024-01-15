FROM golang:1.21 AS build

ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_ENABLED=0

WORKDIR /socket-exporter
COPY . /socket-exporter

RUN --mount=type=cache,target=/root/.cache/go-build,sharing=private \
  go build -o bin/socket-exporter .

# ---
FROM scratch AS run
# Switch for debugging
# FROM ubuntu:latest

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /socket-exporter/bin/socket-exporter /usr/local/bin/

CMD ["socket-exporter"]