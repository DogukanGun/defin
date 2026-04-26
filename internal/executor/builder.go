package executor

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/dogukangundogan/trader/internal/pool"
	"github.com/dogukangundogan/trader/internal/strategy"
)

var (
	// swapExactTokensForTokens(uint256,uint256,address[],address,uint256) for V2 router
	swapV2Selector = crypto.Keccak256([]byte("swap(uint256,uint256,address,bytes)"))[:4]
	// exactInputSingle((address,address,uint24,address,uint256,uint256,uint256,uint160)) for V3 router
	swapV3Selector = crypto.Keccak256([]byte("exactInputSingle((address,address,uint24,address,uint256,uint256,uint256,uint160))"))[:4]
	// exchange(int128,int128,uint256,uint256) for Curve
	curveExchangeSelector = crypto.Keccak256([]byte("exchange(int128,int128,uint256,uint256)"))[:4]
	// liquidationCall(address,address,address,uint256,bool) for Aave
	liquidationCallSelector = crypto.Keccak256([]byte("liquidationCall(address,address,address,uint256,bool)"))[:4]
)

type Builder struct{}

func NewBuilder() *Builder { return &Builder{} }

func (b *Builder) BuildSwapCalldata(step strategy.SwapStep) ([]byte, error) {
	if step.Pool == nil {
		return nil, fmt.Errorf("step has no pool")
	}

	switch step.Pool.Type() {
	case pool.TypeUniswapV2:
		return b.buildV2Swap(step)
	case pool.TypeUniswapV3:
		return b.buildV3Swap(step)
	case pool.TypeCurve:
		return b.buildCurveSwap(step)
	default:
		return nil, fmt.Errorf("unsupported pool type: %s", step.Pool.Type())
	}
}

func (b *Builder) buildV2Swap(step strategy.SwapStep) ([]byte, error) {
	// Low-level V2 pair swap(uint amount0Out, uint amount1Out, address to, bytes data)
	p := step.Pool
	var amount0Out, amount1Out *big.Int

	if step.TokenIn == p.Token0() {
		// Swapping token0 for token1
		amount0Out = big.NewInt(0)
		out, err := p.GetAmountOut(step.TokenIn, step.AmountIn)
		if err != nil {
			return nil, err
		}
		amount1Out = out
	} else {
		out, err := p.GetAmountOut(step.TokenIn, step.AmountIn)
		if err != nil {
			return nil, err
		}
		amount0Out = out
		amount1Out = big.NewInt(0)
	}

	buf := make([]byte, 4+32*4)
	copy(buf[0:4], swapV2Selector)
	copy(buf[4:36], padUint(amount0Out))
	copy(buf[36:68], padUint(amount1Out))
	// to address would be filled by FlashArb contract
	// data is empty for simple swaps

	return buf, nil
}

func (b *Builder) buildV3Swap(step strategy.SwapStep) ([]byte, error) {
	buf := make([]byte, 4+32*8)
	copy(buf[0:4], swapV3Selector)
	// Simplified: in production, use proper ABI encoding for the tuple
	copy(buf[16:36], step.TokenIn.Bytes())
	copy(buf[48:68], step.TokenOut.Bytes())
	copy(buf[68:100], padUint(big.NewInt(int64(step.Pool.FeeBps()*100)))) // fee in hundredths of bps
	if step.AmountIn != nil {
		copy(buf[164:196], padUint(step.AmountIn))
	}
	return buf, nil
}

func (b *Builder) buildCurveSwap(step strategy.SwapStep) ([]byte, error) {
	// exchange(int128 i, int128 j, uint256 dx, uint256 min_dy)
	var i, j int64
	if step.TokenIn == step.Pool.Token0() {
		i, j = 0, 1
	} else {
		i, j = 1, 0
	}

	buf := make([]byte, 4+32*4)
	copy(buf[0:4], curveExchangeSelector)
	copy(buf[4:36], padUint(big.NewInt(i)))
	copy(buf[36:68], padUint(big.NewInt(j)))
	if step.AmountIn != nil {
		copy(buf[68:100], padUint(step.AmountIn))
	}
	// min_dy = 0 (handled by flash loan atomicity)

	return buf, nil
}

func (b *Builder) BuildLiquidationCalldata(collateral, debt, user common.Address, debtAmount *big.Int) []byte {
	buf := make([]byte, 4+32*5)
	copy(buf[0:4], liquidationCallSelector)
	copy(buf[16:36], collateral.Bytes())
	copy(buf[48:68], debt.Bytes())
	copy(buf[80:100], user.Bytes())
	copy(buf[100:132], padUint(debtAmount))
	buf[163] = 0 // receiveAToken = false

	return buf
}

func padUint(val *big.Int) []byte {
	buf := make([]byte, 32)
	if val != nil {
		b := val.Bytes()
		copy(buf[32-len(b):], b)
	}
	return buf
}

// Unused import guard
var _ common.Address
