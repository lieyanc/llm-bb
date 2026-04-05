package stream

import (
	"sync"

	"llm-bb/internal/model"
)

type Hub struct {
	mu          sync.RWMutex
	nextID      int
	subscribers map[int64]map[int]chan model.Message
}

func NewHub() *Hub {
	return &Hub{
		subscribers: make(map[int64]map[int]chan model.Message),
	}
}

func (h *Hub) Subscribe(roomID int64) (<-chan model.Message, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.nextID++
	id := h.nextID

	if _, ok := h.subscribers[roomID]; !ok {
		h.subscribers[roomID] = make(map[int]chan model.Message)
	}

	ch := make(chan model.Message, 16)
	h.subscribers[roomID][id] = ch

	cancel := func() {
		h.mu.Lock()
		defer h.mu.Unlock()

		roomSubs, ok := h.subscribers[roomID]
		if !ok {
			return
		}

		if stream, ok := roomSubs[id]; ok {
			delete(roomSubs, id)
			close(stream)
		}

		if len(roomSubs) == 0 {
			delete(h.subscribers, roomID)
		}
	}

	return ch, cancel
}

func (h *Hub) Publish(message model.Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, ch := range h.subscribers[message.RoomID] {
		select {
		case ch <- message:
		default:
		}
	}
}
