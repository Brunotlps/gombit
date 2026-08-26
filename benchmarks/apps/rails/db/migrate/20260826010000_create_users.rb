# benchmarks/docs/schema.md "Tables" — users. t.text, not the Rails
# generator default t.string (VARCHAR(255)): the canonical schema's columns
# are unbounded TEXT, matching what benchmarks/apps/gin-gorm's GORM (plain
# `string`, no size tag) and benchmarks/apps/django's TextField both
# generate — a lesson already learned once in benchmarks/apps/django (see
# its README) and applied here from the start instead of relearned.
class CreateUsers < ActiveRecord::Migration[8.1]
  def change
    create_table :users do |t|
      t.text :email, null: false
      t.text :name, null: false
      t.datetime :created_at, null: false
    end
    add_index :users, :email, unique: true
  end
end
