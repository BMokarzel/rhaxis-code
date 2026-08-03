import { Injectable, Logger, NotFoundException } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Repository } from 'typeorm';
import { User } from './user.entity';
import { dispatchToChannel } from '../utils/dispatch';

export interface NotifyPayload {
  type: string;
  orderId: string;
  item: string;
  userId: string;
}

@Injectable()
export class UsersService {
  private readonly logger = new Logger(UsersService.name);

  constructor(
    @InjectRepository(User)
    private readonly userRepository: Repository<User>,
  ) {}

  // Flat, sem branch — GET /users/:id.
  async findById(id: string): Promise<User> {
    const user = await this.userRepository.findOneBy({ id });
    if (!user) {
      throw new NotFoundException(`user ${id} not found`);
    }
    return user;
  }

  async notify(payload: NotifyPayload): Promise<void> {
    let message: string;

    switch (payload.type) {
      case 'order.created':
        message = `your order ${payload.orderId} (${payload.item}) was created`;
        break;
      case 'order.cancelled':
        message = `your order ${payload.orderId} was cancelled`;
        break;
      default:
        message = `update on order ${payload.orderId}`;
    }

    try {
      const user = await this.findById(payload.userId);
      for (const channel of user.notificationChannels) {
        dispatchToChannel(channel, user.id, message);
      }
    } catch (err) {
      this.logger.error(`failed to notify user ${payload.userId}: ${(err as Error).message}`);
    }
  }
}
