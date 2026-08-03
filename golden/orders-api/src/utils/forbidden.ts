import { Response } from 'express';

export function forbidden(res: Response): void {
  res.status(403).json({ error: 'forbidden' });
}
