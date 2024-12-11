ARG TOOL=valheim

FROM golang:1.23 AS build
WORKDIR $GOPATH/github.com/frantjc/sindri
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TOOL=valheim
RUN go build -o /$TOOL ./cmd/$TOOL

FROM debian:stable-slim AS valheim
RUN apt-get update -y \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        lib32gcc-s1 \
        libatomic1 \
        libpulse-dev \
        libpulse0 \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /
ENTRYPOINT ["valheim"]
ARG TOOL=valheim
COPY --from=build /$TOOL /usr/local/bin

FROM scratch AS boil
ARG TOOL=valheim
COPY --from=build /$TOOL /

FROM scratch AS mist
ARG TOOL=valheim
COPY --from=build /$TOOL /

FROM $TOOL
