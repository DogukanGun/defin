package pool

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	mathutil "github.com/dogukangundogan/trader/internal/math"
)

// slot0() selector
var slot0Selector = crypto.Keccak256([]byte("slot0()"))[:4]

// liquidity() selector
var liquiditySelector = crypto.Keccak256([]byte("liquidity()"))[:4]

type UniswapV3Pool struct {
	address common.Address
	chainID int64
	token0  common.Address
	token1  common.Address
	feeBps  int
	mu      sync.RWMutex
	state   *PoolState
}

func NewUniswapV3Pool(address, token0, token1 common.Address, chainID int64, feeBps int) *UniswapV3Pool {
	return &UniswapV3Pool{
		address: address,
		chainID: chainID,
		token0:  token0,
		token1:  token1,
		feeBps:  feeBps,
	}
}

func (p *UniswapV3Pool) Address() common.Address { return p.address }
func (p *UniswapV3Pool) Type() PoolType           { return TypeUniswapV3 }
func (p *UniswapV3Pool) ChainID() int64            { return p.chainID }
func (p *UniswapV3Pool) Token0() common.Address    { return p.token0 }
func (p *UniswapV3Pool) Token1() common.Address    { return p.token1 }
func (p *UniswapV3Pool) FeeBps() int               { return p.feeBps }

func (p *UniswapV3Pool) State() *PoolState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.state.Clone()
}

func (p *UniswapV3Pool) UpdateState(state *PoolState) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.state = state
}

func (p *UniswapV3Pool) GetAmountOut(tokenIn common.Address, amountIn *big.Int) (*big.Int, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.state == nil {
		return nil, fmt.Errorf("pool %s: no state", p.address.Hex())
	}
	if p.state.SqrtPriceX96 == nil || p.state.SqrtPriceX96.Sign() == 0 {
		return nil, fmt.Errorf("pool %s: no price yet", p.address.Hex())
	}
	if p.state.Liquidity == nil || p.state.Liquidity.Sign() == 0 {
		return nil, fmt.Errorf("pool %s: liquidity not fetched yet", p.address.Hex())
	}

	zeroForOne := tokenIn == p.token0
	return mathutil.GetAmountOutV3(amountIn, p.state.SqrtPriceX96, p.state.Liquidity, zeroForOne, p.feeBps)
}

func (p *UniswapV3Pool) GetAmountIn(tokenOut common.Address, amountOut *big.Int) (*big.Int, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.state == nil {
		return nil, fmt.Errorf("pool %s: no state", p.address.Hex())
	}
	if p.state.SqrtPriceX96 == nil || p.state.SqrtPriceX96.Sign() == 0 {
		return nil, fmt.Errorf("pool %s: no price yet", p.address.Hex())
	}
	if p.state.Liquidity == nil || p.state.Liquidity.Sign() == 0 {
		return nil, fmt.Errorf("pool %s: liquidity not fetched yet", p.address.Hex())
	}

	zeroForOne := tokenOut == p.token1
	return mathutil.GetAmountInV3(amountOut, p.state.SqrtPriceX96, p.state.Liquidity, zeroForOne, p.feeBps)
}

// StateCalldata returns slot0() calldata. Liquidity is fetched separately.
func (p *UniswapV3Pool) StateCalldata() []byte {
	return slot0Selector
}

// LiquidityCalldata returns the liquidity() calldata for a separate multicall.
func (p *UniswapV3Pool) LiquidityCalldata() []byte {
	return liquiditySelector
}

func (p *UniswapV3Pool) DecodeState(data []byte) (*PoolState, error) {
	if len(data) < 64 {
		return nil, fmt.Errorf("slot0 response too short: %d bytes", len(data))
	}

	sqrtPriceX96 := new(big.Int).SetBytes(data[0:32])
	// tick is int24 packed in a int256
	tickBig := new(big.Int).SetBytes(data[32:64])
	if tickBig.Bit(255) == 1 {
		tickBig.Sub(tickBig, new(big.Int).Lsh(big.NewInt(1), 256))
	}

	return &PoolState{
		SqrtPriceX96: sqrtPriceX96,
		Tick:         int32(tickBig.Int64()),
	}, nil
}

func (p *UniswapV3Pool) DecodeLiquidity(data []byte) (*big.Int, error) {
	if len(data) < 32 {
		return nil, fmt.Errorf("liquidity response too short")
	}
	return new(big.Int).SetBytes(data[0:32]), nil
}
