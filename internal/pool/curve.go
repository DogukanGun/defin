package pool

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	mathutil "github.com/dogukangundogan/trader/internal/math"
)

var (
	// balances(uint256) selector
	balancesSelector = crypto.Keccak256([]byte("balances(uint256)"))[:4]
	// A() selector
	ampSelector = crypto.Keccak256([]byte("A()"))[:4]
)

type CurvePool struct {
	address  common.Address
	chainID  int64
	tokens   []common.Address
	feeBps   int
	nCoins   int
	mu       sync.RWMutex
	state    *PoolState
}

func NewCurvePool(address common.Address, chainID int64, tokens []common.Address, feeBps int) *CurvePool {
	return &CurvePool{
		address: address,
		chainID: chainID,
		tokens:  tokens,
		feeBps:  feeBps,
		nCoins:  len(tokens),
	}
}

func (p *CurvePool) Address() common.Address { return p.address }
func (p *CurvePool) Type() PoolType           { return TypeCurve }
func (p *CurvePool) ChainID() int64            { return p.chainID }
func (p *CurvePool) Token0() common.Address    { return p.tokens[0] }
func (p *CurvePool) Token1() common.Address    { return p.tokens[1] }
func (p *CurvePool) FeeBps() int               { return p.feeBps }
func (p *CurvePool) Tokens() []common.Address   { return p.tokens }

func (p *CurvePool) State() *PoolState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.state.Clone()
}

func (p *CurvePool) UpdateState(state *PoolState) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.state = state
}

func (p *CurvePool) GetAmountOut(tokenIn common.Address, amountIn *big.Int) (*big.Int, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.state == nil {
		return nil, fmt.Errorf("pool %s: no state", p.address.Hex())
	}

	i, j := p.tokenIndices(tokenIn)
	if i < 0 || j < 0 {
		return nil, fmt.Errorf("token %s not in pool %s", tokenIn.Hex(), p.address.Hex())
	}

	return mathutil.GetDyCurve(p.state.Balances, p.state.AmpFactor, amountIn, i, j, p.nCoins, p.feeBps)
}

func (p *CurvePool) GetAmountIn(tokenOut common.Address, amountOut *big.Int) (*big.Int, error) {
	// Curve doesn't have a clean inverse; use binary search
	return nil, fmt.Errorf("GetAmountIn not implemented for Curve pools")
}

func (p *CurvePool) StateCalldata() []byte {
	// Return calldata for balances(0)
	buf := make([]byte, 36)
	copy(buf[0:4], balancesSelector)
	return buf
}

func (p *CurvePool) AmpCalldata() []byte {
	return ampSelector
}

func (p *CurvePool) DecodeState(data []byte) (*PoolState, error) {
	if len(data) < 32 {
		return nil, fmt.Errorf("balance response too short")
	}
	balance := new(big.Int).SetBytes(data[0:32])
	return &PoolState{
		Balances: []*big.Int{balance},
	}, nil
}

func (p *CurvePool) tokenIndices(tokenIn common.Address) (int, int) {
	idx := -1
	for i, t := range p.tokens {
		if t == tokenIn {
			idx = i
			break
		}
	}
	if idx < 0 {
		return -1, -1
	}
	// Default: return swap to the "other" token (index 0 or 1)
	other := 0
	if idx == 0 {
		other = 1
	}
	return idx, other
}
