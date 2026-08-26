# benchmarks/docs/schema.md "Tables" — projects. t.references without
# on_delete: leaves Postgres's own default (NO ACTION), matching the
# canonical FK exactly — Rails only adds ON DELETE CASCADE/SET NULL/etc. if
# explicitly asked via on_delete:. Also does not pass deferrable:, so the
# constraint is NOT DEFERRABLE (Postgres's own default when omitted) —
# unlike benchmarks/apps/django, whose ORM defaults to DEFERRABLE INITIALLY
# DEFERRED and needed a follow-up migration to match this schema; verified
# here from the start (see ../README.md) rather than discovered as a bug
# later.
class CreateProjects < ActiveRecord::Migration[8.1]
  def change
    create_table :projects do |t|
      t.references :owner, null: false, foreign_key: { to_table: :users }, index: true
      t.text :name, null: false
      t.text :description, null: false, default: ""
      t.timestamps
    end
    add_index :projects, :created_at
  end
end
