# Build context is the repo root (see docker-compose.yml).
FROM golang:1.23-alpine AS build
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/. .
RUN CGO_ENABLED=0 go build -o /api ./cmd/api

FROM alpine:3.20
WORKDIR /app
COPY --from=build /api /app/api
COPY migrations ./migrations
COPY openapi.yaml ./openapi.yaml
EXPOSE 8080
CMD ["/app/api"]
