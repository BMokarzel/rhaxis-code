import { Logger } from '@nestjs/common';

const logger = new Logger('dispatch');

export function dispatchToChannel(channel: string, userId: string, message: string): void {
  switch (channel) {
    case 'email':
      logger.log(`emailing ${userId}: ${message}`);
      break;
    case 'sms':
      logger.log(`sms to ${userId}: ${message}`);
      break;
    case 'push':
      logger.log(`push to ${userId}: ${message}`);
      break;
    default:
      logger.warn(`unknown channel ${channel} for ${userId}`);
  }
}
