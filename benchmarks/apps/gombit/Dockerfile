# Container image for the gombit benchmark app — the real Gombit-runtime
# implementation of the canonical /api/projects CRUD API.
#
# Build context is the REPOSITORY ROOT (the Go module root):
#
#   docker build -f benchmarks/apps/gombit/Dockerfile -t bench-gombit:local .
#
# Unlike gin-gorm (AutoMigrate), this app uses Atlas versioned migrations
# applied as a separate step, exactly like a deployed Gombit app: the entrypoint
# runs `gombit db migrate` (which execs `atlas migrate apply`) before serving.
# So the image carries three things — the app binary, the `gombit` CLI, and the
# pinned `atlas` binary — plus the migrations directory.
ARG ATLAS_IMAGE=arigaio/atlas:0.36.0

FROM golang:1.25.7-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# The app under test and the gombit CLI that applies its migrations.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/gombit-app ./benchmarks/apps/gombit && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/gombit ./cmd/gombit

# Pinned Atlas CLI (distroless image; the binary lives at /atlas).
FROM ${ATLAS_IMAGE} AS atlas

FROM alpine:3.20
RUN adduser -D -u 10001 app
COPY --from=build /out/gombit-app /usr/local/bin/gombit-app
COPY --from=build /out/gombit /usr/local/bin/gombit
COPY --from=atlas /atlas /usr/local/bin/atlas
# `gombit db migrate` defaults --dir to database/migrations relative to the
# working directory, so ship the migrations under /app and run from there.
WORKDIR /app
COPY benchmarks/apps/gombit/database ./database
COPY benchmarks/apps/gombit/docker-entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh
USER app
EXPOSE 8080
# Verbs: `migrate` (atlas apply, then exit), `seed` (deterministic seed, then
# exit), `serve` (run the API, default).
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["serve"]
