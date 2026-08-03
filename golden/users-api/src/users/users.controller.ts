import { Body, Controller, Get, Param, Post, UseGuards } from '@nestjs/common';
import { UsersService, NotifyPayload } from './users.service';
import { ApiKeyGuard } from '../guards/api-key.guard';

@Controller()
export class UsersController {
  constructor(private readonly usersService: UsersService) {}

  // GET /users/:id — flat, sem branch.
  @Get('users/:id')
  async get(@Param('id') id: string) {
    return this.usersService.findById(id);
  }

  // POST /notify — alvo do CallHTTP disparado pelo orders-api.
  // Protegido por ApiKeyGuard (Middleware kind, edge PROTECTS).
  @Post('notify')
  @UseGuards(ApiKeyGuard)
  async notify(@Body() payload: NotifyPayload) {
    await this.usersService.notify(payload);
    return { received: true };
  }
}
