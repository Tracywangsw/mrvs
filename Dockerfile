FROM golang:1.21-alpine AS builder

WORKDIR /src

COPY backend/go.mod backend/go.sum ./backend/
RUN cd backend && go mod download

COPY backend ./backend
RUN cd backend && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /out/server /app/server

ENV APP_PORT=8080
EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --start-period=20s --retries=5 \
  CMD wget -q -O /dev/null http://127.0.0.1:${APP_PORT}/healthz || exit 1

ENTRYPOINT ["/app/server"]
