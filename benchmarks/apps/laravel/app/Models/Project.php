<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class Project extends Model
{
    // Preserve microsecond precision on writes. The created/projects columns
    // are TIMESTAMPTZ(6), but Eloquent's Postgres grammar formats timestamps
    // as 'Y-m-d H:i:s' (whole seconds) by default, so an API-created row's
    // created_at/updated_at would come back as .000000 while the seeded rows
    // (inserted via the query builder with an explicit microsecond format)
    // and every sibling implementation carry real microseconds. 'Y-m-d
    // H:i:s.uP' writes microseconds + tz offset, matching the seeded data and
    // the canonical schema's precision.
    protected $dateFormat = 'Y-m-d H:i:s.uP';

    protected $fillable = ['owner_id', 'name', 'description'];

    public function owner(): BelongsTo
    {
        return $this->belongsTo(User::class, 'owner_id');
    }
}
