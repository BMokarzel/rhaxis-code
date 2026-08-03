import { Request, Response, NextFunction } from 'express';

// Chamada explícita dentro do handler (não é um Guard do Nest) — de propósito,
// para exercitar CallFunction em vez de Middleware/PROTECTS.
export function authMiddleware(req: Request, res: Response, next: NextFunction): void {
  const token = req.headers.authorization ?? '';
  (req as any).user = { isAdmin: token.endsWith('admin'), id: token.replace('Bearer ', '') || 'anonymous' };
  next();
}
