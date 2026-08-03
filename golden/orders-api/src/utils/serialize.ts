import { Order } from '../orders/order.entity';

export function serialize(order: Order) {
  return {
    id: order.id,
    item: order.item,
    qty: order.qty,
    createdAt: order.createdAt.toISOString(),
  };
}
