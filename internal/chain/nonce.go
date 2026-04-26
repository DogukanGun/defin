package chain

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

type NonceTracker struct {
	client *Client
	addr   common.Address
	mu     sync.Mutex
	nonce  uint64
	init   bool
}

func NewNonceTracker(client *Client, addr common.Address) *NonceTracker {
	return &NonceTracker{client: client, addr: addr}
}

func (nt *NonceTracker) Next(ctx context.Context) (uint64, error) {
	nt.mu.Lock()
	defer nt.mu.Unlock()

	if !nt.init {
		pending, err := nt.client.HTTP().PendingNonceAt(ctx, nt.addr)
		if err != nil {
			return 0, err
		}
		nt.nonce = pending
		nt.init = true
	}

	n := nt.nonce
	nt.nonce++
	return n, nil
}

func (nt *NonceTracker) Reset() {
	nt.mu.Lock()
	defer nt.mu.Unlock()
	nt.init = false
}
