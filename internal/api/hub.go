package api

import (
	"encoding/json"
	"sync"
	"time"
)

type EventType string

const (
	EventLog         EventType = "log"
	EventOpportunity EventType = "opportunity"
	EventStatus      EventType = "status"
	EventPool        EventType = "pool"
	EventBalance     EventType = "balance"
)

// EventStep describes one hop in a multi-step trade path.
type EventStep struct {
	TokenIn  string `json:"token_in"`
	TokenOut string `json:"token_out"`
	Pool     string `json:"pool"`
	PoolType string `json:"pool_type"`
}

// TokenBalance is a single token holding for the wallet.
type TokenBalance struct {
	Symbol   string `json:"symbol"`
	Address  string `json:"address"`
	RawWei   string `json:"raw_wei"`
	Decimals int    `json:"decimals"`
}

type Event struct {
	Type          EventType      `json:"type"`
	Level         string         `json:"level,omitempty"`
	Message       string         `json:"message,omitempty"`
	Strategy      string         `json:"strategy,omitempty"`
	NetProfit     string         `json:"net_profit,omitempty"`
	Chain         string         `json:"chain,omitempty"`
	GasPrice      float64        `json:"gas_price_gwei,omitempty"`
	Mode          string         `json:"mode,omitempty"`
	GasEstimate   uint64         `json:"gas_estimate,omitempty"`
	Steps         []EventStep    `json:"steps,omitempty"`
	WalletAddress string         `json:"wallet_address,omitempty"`
	Balances      []TokenBalance `json:"balances,omitempty"`
	Timestamp     time.Time      `json:"timestamp"`
}

type client struct {
	send chan []byte
}

// Hub broadcasts events to all connected WebSocket clients.
type Hub struct {
	mu          sync.RWMutex
	clients     map[*client]struct{}
	lastWallet  string
	lastBals    []TokenBalance
}

func NewHub() *Hub {
	return &Hub{clients: make(map[*client]struct{})}
}

// LastBalance returns the most recently broadcast wallet balance snapshot.
func (h *Hub) LastBalance() (string, []TokenBalance) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.lastWallet, h.lastBals
}

func (h *Hub) register(c *client) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

func (h *Hub) unregister(c *client) {
	h.mu.Lock()
	delete(h.clients, c)
	close(c.send)
	h.mu.Unlock()
}

// Send broadcasts an event to all connected clients.
func (h *Hub) Send(ev Event) {
	ev.Timestamp = time.Now()
	data, err := json.Marshal(ev)
	if err != nil {
		return
	}

	h.mu.Lock()
	if ev.Type == EventBalance {
		h.lastWallet = ev.WalletAddress
		h.lastBals = ev.Balances
	}
	h.mu.Unlock()

	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.clients {
		select {
		case c.send <- data:
		default:
			// Slow client: drop the message rather than block.
		}
	}
}
