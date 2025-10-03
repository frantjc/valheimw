ARG tool=valheimw

FROM golang:1.24 AS build
WORKDIR $GOPATH/github.com/frantjc/valheimw
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG tool=valheimw
RUN CGO_ENABLED=0 go build -o /$tool ./cmd/$tool

FROM debian:stable-slim AS base
RUN apt-get update -y \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        lib32gcc-s1 \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean

FROM base AS valheimw
ENTRYPOINT ["/usr/local/bin/valheimw"]
COPY --from=build /valheimw /usr/local/bin

FROM scratch AS mist
ENTRYPOINT ["/mist"]
COPY --from=build /mist /mist

FROM $tool
