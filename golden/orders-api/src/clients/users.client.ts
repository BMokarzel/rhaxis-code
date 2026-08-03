import { Injectable, Logger } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { HttpService } from '@nestjs/axios';
import { firstValueFrom } from 'rxjs';

// Dependência cross-service: orders-api -> users-api.
@Injectable()
export class UsersClient {
  private readonly logger = new Logger(UsersClient.name);
  private readonly usersApiUrl: string;

  constructor(
    private readonly http: HttpService,
    private readonly config: ConfigService,
  ) {
    this.usersApiUrl = this.config.get<string>('USERS_API_URL', 'http://users-api:3002');
  }

  async notify(orderId: string, item: string, userId: string): Promise<void> {
    this.logger.log(`notifying users-api about order ${orderId}`);
    await firstValueFrom(
      this.http.post(`${this.usersApiUrl}/notify`, {
        type: 'order.created',
        orderId,
        item,
        userId,
      }),
    );
  }
}
