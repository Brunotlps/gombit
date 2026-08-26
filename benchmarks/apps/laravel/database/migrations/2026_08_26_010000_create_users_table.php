<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

// benchmarks/docs/schema.md "Tables" — users. $table->text(), not the Laravel
// generator default $table->string() (VARCHAR(255)): the canonical schema's
// columns are unbounded TEXT, matching what benchmarks/apps/gin-gorm's GORM
// (plain `string`), benchmarks/apps/django's TextField, and
// benchmarks/apps/rails's t.text all generate. timestamptz not needed on
// users (only created_at, and the canonical schema uses TIMESTAMPTZ) — see
// the projects migration for how created_at/updated_at get timestamptz.
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('users', function (Blueprint $table) {
            $table->bigIncrements('id');
            $table->text('email');
            $table->text('name');
            // timestampTz, not timestamp: the canonical schema's created_at is
            // TIMESTAMPTZ (verified against gin-gorm/gombit's real migrations).
            $table->timestampTz('created_at', 6);
            $table->unique('email');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('users');
    }
};
