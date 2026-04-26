package pool

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

type Registry struct {
	mu       sync.RWMutex
	byAddr   map[common.Address]Pool
	byPair   map[pairKey][]Pool
	byToken  map[common.Address][]Pool
	byChain  map[int64][]Pool
}

type pairKey struct {
	token0  common.Address
	token1  common.Address
	chainID int64
}

func NewRegistry() *Registry {
	return &Registry{
		byAddr:  make(map[common.Address]Pool),
		byPair:  make(map[pairKey][]Pool),
		byToken: make(map[common.Address][]Pool),
		byChain: make(map[int64][]Pool),
	}
}

func (r *Registry) Add(p Pool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	addr := p.Address()
	if _, exists := r.byAddr[addr]; exists {
		return
	}

	r.byAddr[addr] = p

	t0, t1 := p.Token0(), p.Token1()
	pk := makePairKey(t0, t1, p.ChainID())
	r.byPair[pk] = append(r.byPair[pk], p)

	r.byToken[t0] = append(r.byToken[t0], p)
	r.byToken[t1] = append(r.byToken[t1], p)

	r.byChain[p.ChainID()] = append(r.byChain[p.ChainID()], p)
}

func (r *Registry) Get(addr common.Address) (Pool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.byAddr[addr]
	return p, ok
}

func (r *Registry) GetByPair(token0, token1 common.Address, chainID int64) []Pool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	pk := makePairKey(token0, token1, chainID)
	return r.byPair[pk]
}

func (r *Registry) GetByToken(token common.Address) []Pool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.byToken[token]
}

func (r *Registry) GetByChain(chainID int64) []Pool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.byChain[chainID]
}

func (r *Registry) All() []Pool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	pools := make([]Pool, 0, len(r.byAddr))
	for _, p := range r.byAddr {
		pools = append(pools, p)
	}
	return pools
}

func (r *Registry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.byAddr)
}

func makePairKey(a, b common.Address, chainID int64) pairKey {
	if a.Hex() > b.Hex() {
		a, b = b, a
	}
	return pairKey{token0: a, token1: b, chainID: chainID}
}
