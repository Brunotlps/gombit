<?php

namespace Database\Seeders;

use App\Support\CanonicalSeed;
use Illuminate\Database\Seeder;

// Seed the deterministic BENCH-1 dataset (1,000 users, 100,000 projects),
// truncating first. Run with `php artisan db:seed --force`.
class DatabaseSeeder extends Seeder
{
    public function run(): void
    {
        CanonicalSeed::seedDatabase();
        $this->command->info('laravel: seed complete');
    }
}
