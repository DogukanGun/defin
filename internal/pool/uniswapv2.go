package pool

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	mathutil "github.com/dogukangundogan/trader/internal/math"
)

// getReserves() selector
var getReservesSelector = crypto.Keccak256([]byte("getReserves()"))[:4]

type UniswapV2Pool struct {
	address common.Address
	chainID int64
	token0  common.Address
	token1  common.Address
	feeBps  int
	mu      sync.RWMutex
	state   *PoolState
}

func NewUniswapV2Pool(address, token0, token1 common.Address, chainID int64, feeBps int) *UniswapV2Pool {
	return &UniswapV2Pool{
		address: address,
		chainID: chainID,
		token0:  token0,
		token1:  token1,
		feeBps:  feeBps,
	}
}

func (p *UniswapV2Pool) Address() common.Address { return p.address }
func (p *UniswapV2Pool) Type() PoolType           { return TypeUniswapV2 }
func (p *UniswapV2Pool) ChainID() int64            { return p.chainID }
func (p *UniswapV2Pool) Token0() common.Address    { return p.token0 }
func (p *UniswapV2Pool) Token1() common.Address    { return p.token1 }
func (p *UniswapV2Pool) FeeBps() int               { return p.feeBps }

func (p *UniswapV2Pool) State() *PoolState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.state.Clone()
}

func (p *UniswapV2Pool) UpdateState(state *PoolState) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.state = state
}

func (p *UniswapV2Pool) GetAmountOut(tokenIn common.Address, amountIn *big.Int) (*big.Int, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.state == nil {
		return nil, fmt.Errorf("pool %s: no state", p.address.Hex())
	}

	var reserveIn, reserveOut *big.Int
	if tokenIn == p.token0 {
		reserveIn = p.state.Reserve0
		reserveOut = p.state.Reserve1
	} else if tokenIn == p.token1 {
		reserveIn = p.state.Reserve1
		reserveOut = p.state.Reserve0
	} else {
		return nil, fmt.Errorf("token %s not in pool %s", tokenIn.Hex(), p.address.Hex())
	}

	return mathutil.GetAmountOutV2(amountIn, reserveIn, reserveOut, p.feeBps), nil
}

func (p *UniswapV2Pool) GetAmountIn(tokenOut common.Address, amountOut *big.Int) (*big.Int, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.state == nil {
		return nil, fmt.Errorf("pool %s: no state", p.address.Hex())
	}

	var reserveIn, reserveOut *big.Int
	if tokenOut == p.token1 {
		reserveIn = p.state.Reserve0
		reserveOut = p.state.Reserve1
	} else if tokenOut == p.token0 {
		reserveIn = p.state.Reserve1
		reserveOut = p.state.Reserve0
	} else {
		return nil, fmt.Errorf("token %s not in pool %s", tokenOut.Hex(), p.address.Hex())
	}

	return mathutil.GetAmountInV2(amountOut, reserveIn, reserveOut, p.feeBps), nil
}

func (p *UniswapV2Pool) StateCalldata() []byte {
	return getReservesSelector
}

func (p *UniswapV2Pool) DecodeState(data []byte) (*PoolState, error) {
	if len(data) < 96 {
		return nil, fmt.Errorf("getReserves response too short: %d bytes", len(data))
	}

	reserve0 := new(big.Int).SetBytes(data[0:32])
	reserve1 := new(big.Int).SetBytes(data[32:64])
	// data[64:96] is blockTimestampLast, ignored

	return &PoolState{
		Reserve0: reserve0,
		Reserve1: reserve1,
	}, nil
}
