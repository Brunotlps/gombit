# Container image for the django benchmark app.
#
# The app is self-contained (no Go module), so the build context is THIS
# directory, not the repo root:
#
#   docker build -t bench-django:local benchmarks/apps/django
#
# benchmarks/compose.yml builds it with `context: apps/django`. Migrations are
# applied as a separate step (like a deployed Django app); `serve` runs gunicorn
# (WSGI, issue #141 §17), never `manage.py runserver`.
FROM python:3.12-slim
ENV PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1 \
    PIP_NO_CACHE_DIR=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1
WORKDIR /app
# Install deps first so app-source edits don't re-run pip.
COPY requirements.txt ./
RUN pip install -r requirements.txt
COPY . ./
RUN useradd -u 10001 -m app
USER app
EXPOSE 8082
# Verbs: migrate (ensure the database + apply migrations), seed, serve
# (gunicorn, default). serve does NOT migrate — run migrate first.
ENTRYPOINT ["/app/docker-entrypoint.sh"]
CMD ["serve"]
