ARG tool=valheimw

FROM golang:1.24 AS build
WORKDIR $GOPATH/github.com/frantjc/sindri
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

FROM base AS boiler
ENTRYPOINT ["/usr/local/bin/boiler"]
COPY --from=build /boiler /usr/local/bin

FROM scratch AS mist
ENTRYPOINT ["/mist"]
COPY --from=build /mist /mist

FROM node:20.11.1-slim AS remix
WORKDIR /src/github.com/frantjc/sindri
COPY package.json yarn.lock ./
RUN yarn
COPY app/ app/
COPY public/ public/
COPY *.js *.ts tsconfig.json ./
RUN yarn build

FROM base AS stoker
ENTRYPOINT ["/usr/local/bin/stoker"]
COPY --from=build /stoker /usr/local/bin

FROM $tool
