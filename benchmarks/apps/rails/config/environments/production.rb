require "active_support/core_ext/integer/time"

Rails.application.configure do
  # Settings specified here will take precedence over those in config/application.rb.

  # Code is not reloaded between requests.
  config.enable_reloading = false

  # Eager load code on boot for better performance and memory savings (ignored by Rake tasks).
  config.eager_load = true

  # Full error reports are disabled.
  config.consider_all_requests_local = false

  # Cache assets for far-future expiry since they are all digest stamped.
  config.public_file_server.headers = { "cache-control" => "public, max-age=#{1.year.to_i}" }

  # Enable serving of images, stylesheets, and JavaScripts from an asset server.
  # config.asset_host = "http://assets.example.com"

  # No TLS-terminating reverse proxy in front of this benchmark app — every
  # other benchmarks/apps/ implementation is also served plain HTTP
  # directly (issue #141's benchmark harness talks to each app over plain
  # HTTP; TLS termination is out of scope for what's being measured here).
  # Rails' scaffolded default (assume_ssl/force_ssl both true) would 301 or
  # reject every request from a plain-HTTP client, including the fairness
  # check and `curl`.
  config.assume_ssl = false
  config.force_ssl = false

  # Skip http-to-https redirect for the default health check endpoint.
  # config.ssl_options = { redirect: { exclude: ->(request) { request.path == "/up" } } }

  # Log to STDOUT with the current request id as a default log tag.
  config.log_tags = [ :request_id ]
  config.logger   = ActiveSupport::TaggedLogging.logger(STDOUT)

  # Change to "debug" to log everything (including potentially personally-identifiable information!).
  config.log_level = ENV.fetch("RAILS_LOG_LEVEL", "info")

  # Prevent health checks from clogging up the logs.
  config.silence_healthcheck_path = "/up"

  # Don't log any deprecations.
  config.active_support.report_deprecations = false

  # Replace the default in-process memory cache store with a durable alternative.
  # config.cache_store = :mem_cache_store

  # Replace the default in-process and non-durable queuing backend for Active Job.
  # config.active_job.queue_adapter = :resque

  # Enable locale fallbacks for I18n (makes lookups for any locale fall back to
  # the I18n.default_locale when a translation cannot be found).
  config.i18n.fallbacks = true

  # Do not dump schema after migrations.
  config.active_record.dump_schema_after_migration = false

  # Only use :id for inspections in production.
  config.active_record.attributes_for_inspect = [ :id ]

  # RAILS_ALLOWED_HOSTS, comma-separated (default 127.0.0.1,localhost) —
  # matches benchmarks/apps/django's DJANGO_ALLOWED_HOSTS convention. Rails'
  # DNS-rebinding protection (config.hosts) rejects any request whose Host
  # header isn't in this list with a 403, before it ever reaches routing —
  # needed for local curl/fairness-check testing against 127.0.0.1.
  config.hosts = ENV.fetch("RAILS_ALLOWED_HOSTS", "127.0.0.1,localhost").split(",")
end
