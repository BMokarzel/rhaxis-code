import { Column, CreateDateColumn, Entity, PrimaryGeneratedColumn } from 'typeorm';

@Entity({ name: 'orders' })
export class Order {
  @PrimaryGeneratedColumn('uuid')
  id: string;

  @Column()
  item: string;

  @Column('int')
  qty: number;

  @Column({ default: false })
  placedByAdmin: boolean;

  @CreateDateColumn()
  createdAt: Date;
}
