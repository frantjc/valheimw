ARG tool=valheimw

FROM golang:1.23 AS build
WORKDIR $GOPATH/github.com/frantjc/sindri
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG tool=valheimw
RUN CGO_ENABLED=0 go build -o /$tool ./cmd/$tool

FROM debian:stable-slim AS valheimw
RUN apt-get update -y \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        lib32gcc-s1 \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean
ENTRYPOINT ["/usr/local/bin/valheimw"]
COPY --from=build /valheimw /usr/local/bin

FROM debian:stable-slim AS boiler
RUN apt-get update -y \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        lib32gcc-s1 \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean
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

FROM node:20.11.1-slim AS stoker
RUN apt-get update -y \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        lib32gcc-s1 \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean
ENV NODE_ENV production
ENTRYPOINT ["/usr/local/bin/stoker"]
COPY server.js package.json /app/
COPY --from=build /stoker /usr/local/bin/stoker
COPY --from=remix /src/github.com/frantjc/sindri/build /app/build/
COPY --from=remix /src/github.com/frantjc/sindri/node_modules /app/node_modules/
COPY --from=remix /src/github.com/frantjc/sindri/public /app/public/

FROM $tool
