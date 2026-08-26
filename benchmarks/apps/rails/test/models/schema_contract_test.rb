require "test_helper"

# benchmarks/docs/schema.md requires TIMESTAMPTZ columns and an
# immediately-checked (non-deferrable) foreign key. Both properties depend
# on config/initializers/datetime_type.rb's load hook firing and
# db/migrate/*_create_projects.rb's t.references never gaining a
# deferrable: option — verified only by hand (`psql \d`) during development
# until this test existed. A hook that silently stops firing (e.g. a future
# Rails upgrade renaming the :active_record_postgresqladapter load hook)
# would still pass every CRUD/list/N+1 test in this suite on plain
# `timestamp without time zone` columns without this check — the same
# "documented, not pinned" gap review found in benchmarks/apps/django's
# schema before its own follow-up migration.
class SchemaContractTest < ActiveSupport::TestCase
  TIMESTAMPTZ_COLUMNS = {
    "users" => %w[created_at],
    "projects" => %w[created_at updated_at]
  }.freeze

  test "created_at and updated_at are timestamptz, not the Postgres adapter's plain timestamp default" do
    connection = ActiveRecord::Base.connection

    TIMESTAMPTZ_COLUMNS.each do |table, columns|
      columns.each do |column|
        row = connection.select_one(<<~SQL)
          SELECT data_type, udt_name FROM information_schema.columns
          WHERE table_name = #{connection.quote(table)} AND column_name = #{connection.quote(column)}
        SQL
        assert row, "#{table}.#{column} column not found"
        assert_equal "timestamp with time zone", row["data_type"], "#{table}.#{column} data_type"
        assert_equal "timestamptz", row["udt_name"], "#{table}.#{column} udt_name"
      end
    end
  end

  test "projects.owner_id foreign key is immediately checked, not deferrable" do
    connection = ActiveRecord::Base.connection
    row = connection.select_one(<<~SQL)
      SELECT condeferrable, condeferred FROM pg_constraint
      WHERE conrelid = 'projects'::regclass AND contype = 'f'
    SQL
    assert row, "projects.owner_id foreign key constraint not found"
    assert_not row["condeferrable"], "projects.owner_id FK must not be DEFERRABLE"
    assert_not row["condeferred"], "projects.owner_id FK must not be INITIALLY DEFERRED"
  end
end
