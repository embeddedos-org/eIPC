// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package core

import (
	"container/heap"
	"sync"
)

// HandlerFunc processes an incoming EIPC message and optionally returns a response.
type HandlerFunc func(msg Message) (*Message, error)

// Router dispatches incoming messages to registered handlers by message type.
// It supports priority-lane awareness: P0 messages are dispatched first.
type Router struct {
	mu       sync.RWMutex
	handlers map[MessageType]HandlerFunc
}

// NewRouter creates a new message router.
func NewRouter() *Router {
	return &Router{
		handlers: make(map[MessageType]HandlerFunc),
	}
}

// Handle registers a handler for the given message type.
func (r *Router) Handle(msgType MessageType, handler HandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[msgType] = handler
}

// Dispatch routes a single message to its registered handler.
//
// Dispatch performs no identity check: any message reaches its handler.
// That is the unsecured/local path. Secured servers must use DispatchFor
// with the peer bound to the connection instead.
func (r *Router) Dispatch(msg Message) (*Message, error) {
	r.mu.RLock()
	handler, ok := r.handlers[msg.Type]
	r.mu.RUnlock()

	if !ok {
		return nil, nil
	}
	return handler(msg)
}

// DispatchFor validates msg against the authenticated peer before
// dispatching: a forged Source or SessionID, or an ungranted Capability,
// never reaches a handler. A nil peer behaves exactly like Dispatch
// (unsecured/local path).
func (r *Router) DispatchFor(msg Message, peer *AuthenticatedPeer) (*Message, error) {
	if err := peer.ValidateMessage(msg); err != nil {
		return nil, err
	}
	return r.Dispatch(msg)
}

// DispatchBatch processes a batch of messages in priority order (P0 first).
// Messages with the same priority are dispatched in the order they appear
// in the batch.
func (r *Router) DispatchBatch(messages []Message) []DispatchResult {
	return r.dispatchBatch(messages, r.Dispatch)
}

// DispatchBatchFor is DispatchBatch with per-message identity binding:
// each message is validated against the authenticated peer before its
// handler runs. A nil peer behaves exactly like DispatchBatch.
func (r *Router) DispatchBatchFor(messages []Message, peer *AuthenticatedPeer) []DispatchResult {
	return r.dispatchBatch(messages, func(msg Message) (*Message, error) {
		return r.DispatchFor(msg, peer)
	})
}

func (r *Router) dispatchBatch(messages []Message, dispatch func(Message) (*Message, error)) []DispatchResult {
	pq := make(priorityQueue, len(messages))
	for i, msg := range messages {
		pq[i] = &priorityItem{msg: msg, index: i}
	}
	heap.Init(&pq)

	results := make([]DispatchResult, len(messages))
	for pq.Len() > 0 {
		item := heap.Pop(&pq).(*priorityItem)
		resp, err := dispatch(item.msg)
		results[item.index] = DispatchResult{Response: resp, Err: err}
	}
	return results
}

// DispatchResult holds the outcome of dispatching a single message.
type DispatchResult struct {
	Response *Message
	Err      error
}

// Priority queue implementation for priority-lane dispatch.

type priorityItem struct {
	msg   Message
	index int
}

type priorityQueue []*priorityItem

func (pq priorityQueue) Len() int { return len(pq) }

// Less orders by priority (lower value = higher priority), then by position
// in the batch. container/heap is not stable, so without the second key
// messages that share a priority lane come out in an arbitrary order.
func (pq priorityQueue) Less(i, j int) bool {
	if pq[i].msg.Priority != pq[j].msg.Priority {
		return pq[i].msg.Priority < pq[j].msg.Priority
	}
	return pq[i].index < pq[j].index
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *priorityQueue) Push(x interface{}) {
	*pq = append(*pq, x.(*priorityItem))
}

func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	*pq = old[:n-1]
	return item
}
