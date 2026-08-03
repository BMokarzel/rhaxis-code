import { Injectable, NotFoundException } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Repository } from 'typeorm';
import { Order } from './order.entity';

@Injectable()
export class OrdersService {
  constructor(
    @InjectRepository(Order)
    private readonly orderRepository: Repository<Order>,
  ) {}

  // Flat, sem branch — GET /orders/:id.
  async findById(id: string): Promise<Order> {
    const order = await this.orderRepository.findOneBy({ id });
    if (!order) {
      throw new NotFoundException(`order ${id} not found`);
    }
    return order;
  }

  async insertOrder(item: string, qty: number, placedByAdmin: boolean): Promise<Order> {
    const order = this.orderRepository.create({ item, qty, placedByAdmin });
    return this.orderRepository.save(order);
  }
}
