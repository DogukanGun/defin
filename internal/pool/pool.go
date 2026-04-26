package pool

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type PoolType string

const (
	TypeUniswapV2  PoolType = "uniswap_v2"
	TypeUniswapV3  PoolType = "uniswap_v3"
	TypeCurve      PoolType = "curve"
)

type Pool interface {
	Address() common.Address
	Type() PoolType
	ChainID() int64
	Token0() common.Address
	Token1() common.Address
	FeeBps() int
	State() *PoolState
	UpdateState(state *PoolState)
	GetAmountOut(tokenIn common.Address, amountIn *big.Int) (*big.Int, error)
	GetAmountIn(tokenOut common.Address, amountOut *big.Int) (*big.Int, error)
	// StateCalldata returns the calldata to fetch this pool's state via multicall.
	StateCalldata() []byte
	// DecodeState decodes the multicall response into a PoolState.
	DecodeState(data []byte) (*PoolState, error)
}

type PoolState struct {
	Reserve0     *big.Int // V2: reserve0, V3: liquidity
	Reserve1     *big.Int // V2: reserve1, V3: sqrtPriceX96
	BlockNumber  uint64
	// V3-specific
	SqrtPriceX96 *big.Int
	Tick         int32
	Liquidity    *big.Int
	// Curve-specific
	Balances     []*big.Int
	AmpFactor    *big.Int
}

func (s *PoolState) Clone() *PoolState {
	if s == nil {
		return nil
	}
	c := &PoolState{
		BlockNumber: s.BlockNumber,
		Tick:        s.Tick,
	}
	if s.Reserve0 != nil {
		c.Reserve0 = new(big.Int).Set(s.Reserve0)
	}
	if s.Reserve1 != nil {
		c.Reserve1 = new(big.Int).Set(s.Reserve1)
	}
	if s.SqrtPriceX96 != nil {
		c.SqrtPriceX96 = new(big.Int).Set(s.SqrtPriceX96)
	}
	if s.Liquidity != nil {
		c.Liquidity = new(big.Int).Set(s.Liquidity)
	}
	if s.AmpFactor != nil {
		c.AmpFactor = new(big.Int).Set(s.AmpFactor)
	}
	if s.Balances != nil {
		c.Balances = make([]*big.Int, len(s.Balances))
		for i, b := range s.Balances {
			c.Balances[i] = new(big.Int).Set(b)
		}
	}
	return c
}
