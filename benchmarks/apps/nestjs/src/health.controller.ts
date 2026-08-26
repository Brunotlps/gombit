import { Controller, Get, HttpCode } from '@nestjs/common';

// /livez, matching benchmarks/apps/gin-gorm/gombit/rails/laravel and what
// benchmarks/apps/fairness_test.go's waitForHealth polls. Registered outside
// the global `api` prefix (see main.ts) so it stays at /livez, not
// /api/livez.
@Controller()
export class HealthController {
  @Get('livez')
  @HttpCode(200)
  livez(): void {}
}
