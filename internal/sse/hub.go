package sse

import (
	"sync"
)

// Hub manages channels per job id
type Hub struct {
	mu    sync.RWMutex
	chans map[string]*Client
}

// Client holds a channel where messages for a job are pushed
type Client struct {
	Ch chan string
}

func NewHub() *Hub {
	return &Hub{chans: make(map[string]*Client)}
}

func (h *Hub) Run() {
	// placeholder for expansion (cleanups, TTLs etc.)
}

func (h *Hub) Create(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.chans[id]; !ok {
		h.chans[id] = &Client{Ch: make(chan string, 100)}
	}
}

func (h *Hub) AddClient(id string) *Client {
	h.mu.Lock()
	defer h.mu.Unlock()
	c, ok := h.chans[id]
	if !ok {
		c = &Client{Ch: make(chan string, 100)}
		h.chans[id] = c
	}
	return c
}

func (h *Hub) RemoveClient(id string, client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	close(client.Ch)
	delete(h.chans, id)
}

func (h *Hub) Send(id, msg string) error {
	h.mu.RLock()
	defer h.mu.RUnlock()
	c, ok := h.chans[id]
	if !ok {
		// drop silently or return error
		return nil
	}
	select {
	case c.Ch <- msg:
		return nil
	default:
		// backpressure: drop
		return nil
	}
}
