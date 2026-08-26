<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\HasMany;

// A plain Eloquent Model, not the scaffolded Authenticatable: this benchmark
// has no auth (issue #141 excludes cross-framework auth from the CRUD apps),
// and the canonical users table has only id/email/name/created_at — no
// password, remember_token, or email_verified_at.
class User extends Model
{
    // Only created_at exists on users (no updated_at) — the canonical schema
    // gives users a single created_at column. UPDATED_AT = null tells
    // Eloquent not to expect/maintain an updated_at.
    const UPDATED_AT = null;

    // Microsecond precision on writes — see Project's $dateFormat comment.
    protected $dateFormat = 'Y-m-d H:i:s.uP';

    protected $fillable = ['email', 'name'];

    public function projects(): HasMany
    {
        return $this->hasMany(Project::class, 'owner_id');
    }
}
