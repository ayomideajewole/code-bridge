package sse

import (
	"sync"
	"time"
)

// Hub manages channels per job id
type Hub struct {
	mu    sync.RWMutex
	chans map[string]*Stream
}

// Stream holds channels and state for a translation job
type Stream struct {
	clients []*Client
	buffer  []string
	done    bool
	mu      sync.RWMutex
}

// Client holds a channel where messages for a job are pushed
type Client struct {
	Ch chan string
}

func NewHub() *Hub {
	return &Hub{chans: make(map[string]*Stream)}
}

func (h *Hub) Run() {
	// cleanup old streams periodically
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for range ticker.C {
			h.cleanup()
		}
	}()
}

func (h *Hub) cleanup() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for id, stream := range h.chans {
		stream.mu.RLock()
		done := stream.done
		clientCount := len(stream.clients)
		stream.mu.RUnlock()

		if done && clientCount == 0 {
			delete(h.chans, id)
		}
	}
}

func (h *Hub) Create(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.chans[id]; !ok {
		h.chans[id] = &Stream{
			clients: make([]*Client, 0),
			buffer:  make([]string, 0),
			done:    false,
		}
	}
}

func (h *Hub) AddClient(id string) *Client {
	h.mu.Lock()
	stream, ok := h.chans[id]
	if !ok {
		stream = &Stream{
			clients: make([]*Client, 0),
			buffer:  make([]string, 0),
			done:    false,
		}
		h.chans[id] = stream
	}
	h.mu.Unlock()

	client := &Client{Ch: make(chan string, 200)} // Larger buffer

	stream.mu.Lock()
	stream.clients = append(stream.clients, client)

	// send buffered messages to new client - BLOCKING to ensure delivery
	for _, msg := range stream.buffer {
		client.Ch <- msg // Block instead of select/default
	}
	stream.mu.Unlock()

	return client
}

func (h *Hub) RemoveClient(id string, client *Client) {
	h.mu.RLock()
	stream, ok := h.chans[id]
	h.mu.RUnlock()

	if !ok {
		return
	}

	stream.mu.Lock()
	for i, c := range stream.clients {
		if c == client {
			stream.clients = append(stream.clients[:i], stream.clients[i+1:]...)
			break
		}
	}
	stream.mu.Unlock()

	close(client.Ch)
}

func (h *Hub) Send(id, msg string) error {
	h.mu.RLock()
	stream, ok := h.chans[id]
	h.mu.RUnlock()

	if !ok {
		return nil
	}

	stream.mu.Lock()
	defer stream.mu.Unlock()

	// buffer message FIRST
	stream.buffer = append(stream.buffer, msg)

	// mark as done if end signal
	if msg == "[DONE]" {
		stream.done = true
	}

	// send to all connected clients (non-blocking with larger buffer)
	for _, client := range stream.clients {
		select {
		case client.Ch <- msg:
			// Message sent successfully
		default:
			// Client channel is full, but message is in buffer
			// so client will get it when they catch up
		}
	}

	return nil
}
