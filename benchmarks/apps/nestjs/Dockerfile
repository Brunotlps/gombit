# Container image for the nestjs benchmark app.
#
# Self-contained (no Go module), so the build context is THIS directory:
#
#   docker build -t bench-nestjs:local benchmarks/apps/nestjs
#
# benchmarks/compose.yml builds it with `context: apps/nestjs`. `serve` runs the
# compiled output (`node dist/main`, issue #141 §17), never a ts-node/watch dev
# server. migrate/seed use the committed npm scripts, which drive TypeORM via
# ts-node — so devDependencies are kept in the image (NODE_ENV is set at run
# time by compose, NOT here: setting it to production would make `npm ci` skip
# the devDependencies those scripts need).
FROM node:24-slim
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . ./
RUN npm run build && chown -R node:node /app
USER node
EXPOSE 8085
# Verbs: migrate (ensure the database + TypeORM migration:run), seed, serve
# (node dist/main, default). serve does NOT migrate — run migrate first.
ENTRYPOINT ["/app/docker-entrypoint.sh"]
CMD ["serve"]
