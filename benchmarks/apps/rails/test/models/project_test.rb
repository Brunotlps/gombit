require "test_helper"

class ProjectTest < ActiveSupport::TestCase
  setup do
    @owner = User.create!(email: "owner@example.com", name: "Owner")
  end

  test "rejects blank or whitespace-only name" do
    ["", "   "].each do |name|
      project = Project.new(owner: @owner, name: name, description: "x")
      assert_not project.valid?, "name = #{name.inspect} should be invalid"
      assert_includes project.errors[:name], "can't be blank"
    end
  end

  # belongs_to's default required-association validation rejects this for
  # free (no User has id 0) — the same case
  # benchmarks/apps/gin-gorm's binding:"required" and
  # benchmarks/apps/django's serializer min_value=1 needed dedicated code
  # for. See app/models/project.rb's comment.
  test "rejects owner_id zero" do
    project = Project.new(owner_id: 0, name: "x", description: "y")
    assert_not project.valid?
    assert_includes project.errors[:owner], "must exist"
  end

  test "requires an existing owner" do
    project = Project.new(owner_id: 999_999, name: "x", description: "y")
    assert_not project.valid?
    assert_includes project.errors[:owner], "must exist"
  end
end
