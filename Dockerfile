FROM node:22-alpine AS frontend-builder

WORKDIR /app

RUN corepack enable

COPY package.json pnpm-lock.yaml ./
RUN pnpm config set dangerously-allow-all-builds true
RUN pnpm install --frozen-lockfile

COPY . .
RUN VITE_API_BASE_URL= pnpm exec vite build --outDir dist

FROM golang:1.24-alpine AS backend-builder

WORKDIR /app/mibo-server

COPY mibo-server/go.mod mibo-server/go.sum ./
RUN go mod download

COPY mibo-server/ ./
RUN rm -rf internal/webui/dist && mkdir -p internal/webui/dist
COPY --from=frontend-builder /app/dist/ ./internal/webui/dist/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/mibo-media-server ./cmd/mibo-media-server

FROM alpine:3.22 AS runtime

RUN apk add --no-cache ca-certificates tzdata ffmpeg

WORKDIR /app

ENV MIBO_HTTP_ADDR=:18081
ENV MIBO_DATABASE_DSN=/data/mibo.db

COPY --from=backend-builder /out/mibo-media-server /usr/local/bin/mibo-media-server

VOLUME ["/data", "/media"]

EXPOSE 18081

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 CMD wget -qO- http://127.0.0.1:18081/healthz >/dev/null || exit 1

ENTRYPOINT ["/usr/local/bin/mibo-media-server"]
