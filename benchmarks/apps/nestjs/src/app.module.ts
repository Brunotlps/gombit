import { Module } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { dataSourceOptions } from './data-source';
import { HealthController } from './health.controller';
import { ProjectModule } from './project/project.module';

@Module({
  imports: [TypeOrmModule.forRoot(dataSourceOptions), ProjectModule],
  controllers: [HealthController],
})
export class AppModule {}
