Rails.application.routes.draw do
  # /livez, not Rails' own default /up (removed, not kept alongside this —
  # see config/environments/production.rb's silence_healthcheck_path, which
  # must name this same path): benchmarks/apps/fairness_test.go's
  # waitForHealth polls /livez, matching benchmarks/apps/gin-gorm and
  # benchmarks/apps/gombit.
  get "livez" => "rails/health#show", as: :livez_check

  namespace :api do
    resources :projects, only: %i[index show create update destroy]
  end
end
