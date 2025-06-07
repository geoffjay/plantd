package main

import (
	"fmt"
)

// Handler defines the type of a callback.
type Handler struct {
	running   bool
	canceled  bool
	callbacks map[string]HandlerCallback
}

// HandlerCallback defines the interface that needs to be implemented.
type HandlerCallback interface {
	Execute(msgBody string) ([]byte, error)
}

// NewHandler creates an instance of a callback.
func NewHandler() *Handler {
	return &Handler{
		running:   false,
		canceled:  false,
		callbacks: make(map[string]HandlerCallback),
	}
}

// AddCallback sets a callback by `name`.
func (h *Handler) AddCallback(name string, callback HandlerCallback) {
	h.callbacks[name] = callback
}

// GetCallback retrieves a callback by `name`.
func (h *Handler) GetCallback(name string) (HandlerCallback, error) {
	if callback, found := h.callbacks[name]; found {
		return callback, nil
	}
	return nil, fmt.Errorf("callback not found for %s", name)
}
