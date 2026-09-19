# syntax=docker/dockerfile:1
# Build context is the repository root, so the image can also see data/*.example.

FROM golang:1.25-bookworm AS build
WORKDIR /src

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server

FROM debian:bookworm-slim
RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates curl \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=build /out/server /app/server
# Seed templates only: resumes and personal data are mounted, never baked in.
COPY data/target_companies.yaml.example data/project_bank.yaml.example /app/seed/
COPY backend/docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/docker-entrypoint.sh \
  && mkdir -p /data /app/resumes

ENV ADDR=:8000 \
    DATA_DIR=/data \
    RESUME_DIR=/app/resumes \
    DASHBOARD_URL=http://localhost:3000

EXPOSE 8000
VOLUME ["/data"]

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD curl -fsS http://localhost:8000/health || exit 1

ENTRYPOINT ["/app/docker-entrypoint.sh"]
