FROM golang:1.21 AS build
WORKDIR $GOPATH/github.com/frantjc/sindri
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /ladon ./cmd/ladon

FROM steamcmd/steamcmd:debian-12
RUN apt-get update -y \
    && apt-get install -y --no-install-recommends \
        xvfb \
        libxi6 \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /
ENTRYPOINT ["/ladon"]
COPY --from=build ladon /
