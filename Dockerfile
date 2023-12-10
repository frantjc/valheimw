# FROM golang:1.21 AS build
# WORKDIR $GOPATH/github.com/frantjc/sindri
# COPY go.mod go.sum ./
# RUN go mod download
# COPY . .
# RUN go build -o /sindri ./cmd/sindri

FROM steamcmd/steamcmd:debian-12
WORKDIR /
ENTRYPOINT ["/sindri"]
CMD ["-s", "/run/sindri", "-r", "/var/lib/sindri"]
# COPY --from=build sindri /
COPY sindri /
