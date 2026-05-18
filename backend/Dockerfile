# syntax=docker/dockerfile:1.6
FROM golang:1.22-alpine AS build
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum* ./
RUN go mod download || true
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" \
    -o /out/orchestrator ./cmd/orchestrator

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/orchestrator /usr/local/bin/orchestrator
COPY migrations /migrations
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/orchestrator"]
