require "test_helper"

# Covers the same contract benchmarks/apps/gin-gorm/main_test.go,
# benchmarks/apps/gombit/internal/project/handler_test.go, and
# benchmarks/apps/django/projects/tests.py pin for their own implementations
# (benchmarks/docs/schema.md "Canonical CRUD API"), so the four suites can't
# silently diverge from each other while each claims to satisfy the same
# contract. Asserts D10 error.code, not just the HTTP status, on every
# rejection test — a lesson from benchmarks/apps/django's review round,
# where a status-only assertion let a wrong-status/right-code bug ship
# unnoticed.
module Api
  class ProjectsControllerTest < ActionDispatch::IntegrationTest
    setup do
      @owner = User.create!(email: "owner@example.com", name: "Owner")
    end

    test "CRUD round trip" do
      post "/api/projects", params: { owner_id: @owner.id, name: "Test Project", description: "desc" }, as: :json
      assert_response :created
      created = JSON.parse(response.body)["data"]
      assert_equal "Test Project", created["name"]
      assert_equal "Owner", created["owner_name"]

      get "/api/projects/#{created['id']}"
      assert_response :ok

      patch "/api/projects/#{created['id']}", params: { name: "Renamed" }, as: :json
      assert_response :ok
      updated = JSON.parse(response.body)["data"]
      assert_equal "Renamed", updated["name"]
      assert_equal "desc", updated["description"] # unchanged

      delete "/api/projects/#{created['id']}"
      assert_response :ok

      get "/api/projects/#{created['id']}"
      assert_response :not_found
    end

    test "rejects blank name on create" do
      ["", "   "].each do |name|
        post "/api/projects", params: { owner_id: @owner.id, name: name, description: "x" }, as: :json
        assert_response :unprocessable_content
        assert_equal "validation_error", JSON.parse(response.body).dig("error", "code")
      end
    end

    test "rejects blank name on update" do
      post "/api/projects", params: { owner_id: @owner.id, name: "Original", description: "desc" }, as: :json
      project_id = JSON.parse(response.body)["data"]["id"]

      ["", "   "].each do |name|
        patch "/api/projects/#{project_id}", params: { name: name }, as: :json
        assert_response :unprocessable_content
        assert_equal "validation_error", JSON.parse(response.body).dig("error", "code")
      end

      # A rejected update must not have partially applied.
      get "/api/projects/#{project_id}"
      assert_equal "Original", JSON.parse(response.body)["data"]["name"]
    end

    test "rejects zero owner_id" do
      post "/api/projects", params: { owner_id: 0, name: "x", description: "y" }, as: :json
      assert_response :unprocessable_content
      assert_equal "validation_error", JSON.parse(response.body).dig("error", "code")
    end

    test "rejects nonexistent owner_id" do
      post "/api/projects", params: { owner_id: 999_999, name: "Orphan", description: "no such owner" }, as: :json
      assert_response :unprocessable_content
      assert_equal "validation_error", JSON.parse(response.body).dig("error", "code")
    end

    test "rejects malformed json" do
      post "/api/projects", params: "{", headers: { "Content-Type" => "application/json" }
      assert_response :unprocessable_content
      assert_equal "validation_error", JSON.parse(response.body).dig("error", "code")
    end

    test "preserves description whitespace on create and update" do
      padded = "  padded  "
      post "/api/projects", params: { owner_id: @owner.id, name: "x", description: padded }, as: :json
      assert_response :created
      assert_equal padded, JSON.parse(response.body)["data"]["description"]

      project_id = JSON.parse(response.body)["data"]["id"]
      patch "/api/projects/#{project_id}", params: { description: "#{padded}2" }, as: :json
      assert_equal "#{padded}2", JSON.parse(response.body)["data"]["description"]
    end

    test "get nonexistent id is not found" do
      get "/api/projects/999999999"
      assert_response :not_found
      assert_equal "not_found", JSON.parse(response.body).dig("error", "code")
    end

    # A route parameter that isn't an id at all must still get the D10
    # not_found envelope, not a raw Postgres type-cast error — see
    # projects_controller.rb's find_project! comment.
    test "get non-numeric id is not found" do
      get "/api/projects/not-a-number"
      assert_response :not_found
      assert_equal "not_found", JSON.parse(response.body).dig("error", "code")
    end

    # Inserts users/projects directly, independent of
    # CanonicalSeed.seed_database_n's full-scale seed, so list tests run in
    # milliseconds — mirrors benchmarks/apps/gin-gorm's seedFixture /
    # benchmarks/apps/django's _seed_fixture. Uses the *actual* ids
    # User.create! returns, not an assumed 1..user_count range: Rails'
    # default transactional tests roll back each test's rows, but Postgres
    # sequences are not transactional, so an earlier test's users can push
    # this test's own past that assumed range — the exact bug
    # benchmarks/apps/django's own list-test fixture had before its review
    # round fixed it, avoided here from the start.
    def seed_fixture(user_count, project_count)
      users = (1..user_count).map { |i| User.create!(email: "fixture-#{i}@example.com", name: "Fixture User #{i}") }
      (1..project_count).each do |i|
        owner = users[CanonicalSeed.project_owner_id(i, user_count) - 1]
        Project.create!(owner: owner, name: "Fixture Project #{i}", description: "fixture")
      end
      users
    end

    test "list pagination and ordering" do
      seed_fixture(3, 25)

      get "/api/projects", params: { page: 1, limit: 20 }
      page1 = JSON.parse(response.body)
      assert_equal({ "page" => 1, "limit" => 20, "total" => 25 }, page1["meta"])
      assert_equal 20, page1["data"].length
      assert_operator page1["data"][0]["id"], :>, page1["data"][1]["id"]
      page1["data"].each { |row| assert_not_equal "", row["owner_name"] }

      get "/api/projects", params: { page: 2, limit: 20 }
      page2 = JSON.parse(response.body)
      assert_equal 5, page2["data"].length
      assert_operator page1["data"][19]["id"], :>, page2["data"][0]["id"]
    end

    # benchmarks/docs/schema.md "Canonical CRUD API": the list endpoint must
    # issue a fixed number of queries independent of page size, not one
    # owner query per row. Project.includes(:owner) (see
    # projects_controller.rb#index) preloads owners via one batched
    # `SELECT ... WHERE id IN (...)`, not a JOIN — verified against real
    # Postgres query logs: 3 queries for a non-empty page (page SELECT,
    # batched owner preload, COUNT), matching benchmarks/apps/gin-gorm's
    # pinned shape exactly (no documented deviation needed here, unlike
    # benchmarks/apps/django's 2-query JOIN strategy).
    test "list does not n+1" do
      seed_fixture(5, 20)
      assert_queries_count(3) do
        get "/api/projects", params: { page: 1, limit: 20 }
      end
      assert_response :ok
    end

    test "list does not n+1 on empty page" do
      seed_fixture(5, 20)
      assert_queries_count(2) do
        get "/api/projects", params: { page: 99, limit: 20 }
      end
      assert_response :ok
    end
  end
end
