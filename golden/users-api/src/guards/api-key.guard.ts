import { CanActivate, ExecutionContext, Injectable, UnauthorizedException } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';

// Aplicado via @UseGuards em POST /notify — chamada interna vinda de outro
// serviço (orders-api), protegida por chave compartilhada.
@Injectable()
export class ApiKeyGuard implements CanActivate {
  constructor(private readonly config: ConfigService) {}

  canActivate(context: ExecutionContext): boolean {
    const req = context.switchToHttp().getRequest();
    const expected = this.config.get<string>('INTERNAL_API_KEY');
    const provided = req.headers['x-api-key'];

    if (provided !== expected) {
      throw new UnauthorizedException('invalid api key');
    }
    return true;
  }
}
