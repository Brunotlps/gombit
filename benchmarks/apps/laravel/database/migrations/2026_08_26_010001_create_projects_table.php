<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

// benchmarks/docs/schema.md "Tables" — projects. The foreign key is created
// with no ->onDelete()/deferrable option, so it stays Postgres's own default
// (NO ACTION, NOT DEFERRABLE) — matching the canonical FK and every sibling.
// benchmarks/apps/django's Django ORM instead defaults Postgres FKs to
// DEFERRABLE INITIALLY DEFERRED and needed a follow-up migration to match;
// Eloquent, like GORM and ActiveRecord, does not, so no such fix is needed
// here (verified via psql \d and SchemaContractTest).
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('projects', function (Blueprint $table) {
            $table->bigIncrements('id');
            $table->unsignedBigInteger('owner_id');
            $table->text('name');
            $table->text('description')->default('');
            // TIMESTAMPTZ, matching the canonical schema; Laravel's
            // $table->timestamps() would emit plain `timestamp` (no tz).
            $table->timestampTz('created_at', 6);
            $table->timestampTz('updated_at', 6);

            $table->foreign('owner_id')->references('id')->on('users');
            $table->index('owner_id');
            $table->index('created_at');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('projects');
    }
};
