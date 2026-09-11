package main

import (
	checkoutpb "github.com/netd-tud/ds-onlineshop/src/notificationservice/genproto/checkout"
)

// OrderQueue represents a fixed-capacity ring buffer for storing completed order results,
// incorporating a deduplication tracking map and circular buffer pointers to manage recent orders efficiently.
type OrderQueue struct {
	orders   []*checkoutpb.OrderResult
	seen     map[string]struct{}
	next     int
	count    int
	capacity int
}

// NewOrderQueue initializes and returns a new OrderQueue with the specified ring buffer capacity
// and an internal map for tracking unique order identifiers.
func NewOrderQueue(capacity int) *OrderQueue {
	return &OrderQueue{
		orders:   make([]*checkoutpb.OrderResult, capacity),
		seen:     make(map[string]struct{}),
		capacity: capacity,
	}
}

// Push adds a new completed order to the ring buffer queue if it has not already been processed.
//
// It checks for duplicate orders using a tracking map, evicts the oldest entry if the queue has reached capacity,
// stores the new order at the current ring buffer index, and updates circular pointers accordingly.
func (q *OrderQueue) Push(order *checkoutpb.OrderResult) {
	orderID := order.GetOrderId()
	if _, exists := q.seen[orderID]; exists {
		return
	}

	log.Info("Adding order to queue: ", orderID)

	if q.count == q.capacity {
		oldOrder := q.orders[q.next]
		if oldOrder != nil {
			delete(q.seen, oldOrder.GetOrderId())
		}
	}

	q.orders[q.next] = order
	q.seen[orderID] = struct{}{}

	q.next = (q.next + 1) % q.capacity
	if q.count < q.capacity {
		q.count++
	}
}

// GetAll retrieves all currently stored orders from the ring buffer queue in chronological order,
// starting from the oldest entry to the most recent one.
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
