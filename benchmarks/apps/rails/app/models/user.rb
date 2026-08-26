class User < ApplicationRecord
  has_many :projects, foreign_key: :owner_id, inverse_of: :owner
end
