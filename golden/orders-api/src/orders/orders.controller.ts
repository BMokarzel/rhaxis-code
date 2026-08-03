import { Body, Controller, Get, Logger, Param, Post, Req, Res, UsePipes, ValidationPipe } from '@nestjs/common';
import type { Request, Response } from 'express';
import { OrdersService } from './orders.service';
import { UsersClient } from '../clients/users.client';
import { CreateOrderDto } from './dto/create-order.dto';
import { authMiddleware } from '../middleware/auth';
import { forbidden } from '../utils/forbidden';
import { serialize } from '../utils/serialize';

@Controller('orders')
export class OrdersController {
  private readonly logger = new Logger(OrdersController.name);

  constructor(
    private readonly ordersService: OrdersService,
    private readonly usersClient: UsersClient,
  ) {}

  // GET /orders/:id — flat, sem branch.
  @Get(':id')
  async get(@Param('id') id: string) {
    const order = await this.ordersService.findById(id);
    return serialize(order);
  }

  // POST /orders — auth inline -> if(isAdmin) -> try{ db insert + http notify + serialize } catch{ log } -> else forbidden.
  @Post()
  @UsePipes(new ValidationPipe({ transform: true }))
  async create(@Body() dto: CreateOrderDto, @Req() req: Request, @Res() res: Response) {
    authMiddleware(req, res, () => undefined);
    const user = (req as any).user as { isAdmin: boolean; id: string };

    if (user.isAdmin) {
      try {
        const order = await this.ordersService.insertOrder(dto.item, dto.qty, true);
        await this.usersClient.notify(order.id, order.item, user.id);
        res.status(201).json(serialize(order));
      } catch (err) {
        this.logger.error(`failed to create order: ${(err as Error).message}`);
        res.status(502).json({ error: 'downstream failure' });
      }
    } else {
      forbidden(res);
    }
  }
}
