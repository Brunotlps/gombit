# Container image for the laravel benchmark app: nginx + PHP-FPM in one
# container (the traditional Laravel production topology, issue #141 §17),
# never `php artisan serve`.
#
# Self-contained (no Go module), so the build context is THIS directory:
#
#   docker build -t bench-laravel:local benchmarks/apps/laravel
#
# benchmarks/compose.yml builds it with `context: apps/laravel`. Migrations are
# applied as a separate step; `serve` runs php-fpm + nginx.
FROM composer:2.8 AS composer

FROM php:8.3-fpm
# install-php-extensions pulls each extension's system libs (libpq for
# pdo_pgsql, oniguruma for mbstring) and cleans up — far less error-prone than
# hand-managing apt build deps. nginx is the web server in front of FPM.
COPY --from=mlocati/php-extension-installer:2.9.0 /usr/bin/install-php-extensions /usr/local/bin/
RUN install-php-extensions pdo_pgsql mbstring zip \
    && apt-get update \
    && apt-get install -y --no-install-recommends nginx \
    && rm -rf /var/lib/apt/lists/*
COPY --from=composer /usr/bin/composer /usr/bin/composer

WORKDIR /app
COPY . ./
# A throwaway APP_KEY lets composer's post-autoload package:discover boot the
# app during the build; the real (still ephemeral) key is set at run time.
RUN APP_KEY=base64:$(head -c 32 /dev/urandom | base64) \
      composer install --no-dev --optimize-autoloader --no-interaction --prefer-dist \
    && chmod -R 0777 storage bootstrap/cache

# Production configs. The FPM pool is installed as zzz-bench.conf, NOT over
# www.conf: PHP-FPM glob-merges all php-fpm.d/*.conf into the one [www] pool,
# and only a file sorting AFTER the base image's docker.conf / zz-docker.conf
# wins the keys they set (access.log, clear_env). deploy/php-fpm.conf restates
# every such key explicitly; installing it last makes the runtime pool the
# committed pool. Also install the nginx vhost and drop the stock :80 site.
COPY deploy/php-fpm.conf /usr/local/etc/php-fpm.d/zzz-bench.conf
COPY deploy/nginx.conf /etc/nginx/conf.d/default.conf
RUN rm -f /etc/nginx/sites-enabled/default

EXPOSE 8084
# Verbs: migrate (ensure the database + apply migrations), seed, serve
# (php-fpm + nginx, default). serve does NOT migrate — run migrate first.
ENTRYPOINT ["/app/docker-entrypoint.sh"]
CMD ["serve"]
