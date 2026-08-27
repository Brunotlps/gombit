# Container image for the rails benchmark app.
#
# Self-contained (no Go module), so the build context is THIS directory:
#
#   docker build -t bench-rails:local benchmarks/apps/rails
#
# benchmarks/compose.yml builds it with `context: apps/rails`. Migrations are
# applied as a separate step; `serve` runs Puma in RAILS_ENV=production
# (clustered, per config/puma.rb), never `bin/rails server` in development mode.
FROM ruby:3.3.12-slim AS build
ENV BUNDLE_WITHOUT="development:test" \
    BUNDLE_FROZEN="1" \
    BUNDLE_JOBS="4"
# Build deps for native gems (pg needs libpq-dev + a compiler).
RUN apt-get update && \
    apt-get install -y --no-install-recommends build-essential libpq-dev git pkg-config && \
    rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY Gemfile Gemfile.lock ./
RUN bundle install
COPY . ./

FROM ruby:3.3.12-slim
ENV BUNDLE_WITHOUT="development:test" \
    BUNDLE_FROZEN="1" \
    RAILS_ENV="production"
# libpq5 is the pg gem's runtime dependency.
RUN apt-get update && \
    apt-get install -y --no-install-recommends libpq5 && \
    rm -rf /var/lib/apt/lists/* && \
    useradd -u 10001 -m app
WORKDIR /app
COPY --from=build /usr/local/bundle /usr/local/bundle
COPY --from=build --chown=app:app /app /app
# tmp/ and log/ are excluded from the build context (dev cruft); Puma/Rails
# create files under them at boot, so make them and hand /app to the app user
# (WORKDIR created the /app dir itself as root).
RUN mkdir -p tmp/pids log && chown -R app:app /app
USER app
EXPOSE 8083
# Verbs: migrate (create the database if needed + migrate), seed, serve
# (Puma, default). serve does NOT migrate — run migrate first.
ENTRYPOINT ["/app/docker-entrypoint.sh"]
CMD ["serve"]
