FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY go.mod .
COPY *.go .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /traefik-merge traefik-merge.go

FROM scratch
COPY --from=builder /traefik-merge /traefik-merge
ENTRYPOINT ["/traefik-merge"]