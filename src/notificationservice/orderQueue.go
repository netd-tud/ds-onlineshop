package main

import checkoutpb "github.com/netd-tud/ds-onlineshop/src/notificationservice/genproto/checkout"

type OrderQueue struct {
	orders   []*checkoutpb.OrderResult
	next     int
	count    int
	capacity int
}

func NewOrderQueue(capacity int) *OrderQueue {
	return &OrderQueue{
		orders:   make([]*checkoutpb.OrderResult, capacity),
		capacity: capacity,
	}
}

func (q *OrderQueue) Push(order *checkoutpb.OrderResult) {
	q.orders[q.next] = order
	q.next = (q.next + 1) % q.capacity
	if q.count < q.capacity {
		q.count++
	}
}

func (q *OrderQueue) GetAll() []*checkoutpb.OrderResult {
	res := make([]*checkoutpb.OrderResult, 0, q.count)

	start := 0
	if q.count == q.capacity {
		start = q.next
	}

	for i := 0; i < q.count; i++ {
		idx := (start + i) % q.capacity
		res = append(res, q.orders[idx])
	}
	return res
}
