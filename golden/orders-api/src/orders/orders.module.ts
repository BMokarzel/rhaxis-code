import { HttpModule } from '@nestjs/axios';
import { Module } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { OrdersController } from './orders.controller';
import { OrdersService } from './orders.service';
import { Order } from './order.entity';
import { UsersClient } from '../clients/users.client';

@Module({
  imports: [TypeOrmModule.forFeature([Order]), HttpModule],
  controllers: [OrdersController],
  providers: [OrdersService, UsersClient],
})
export class OrdersModule {}
