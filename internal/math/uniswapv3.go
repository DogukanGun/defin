package math

import (
	"fmt"
	"math/big"
)

// GetAmountOutV3 calculates output for a single-tick V3 swap (no tick crossing).
// This is an approximation valid when the swap stays within the current tick range.
//
// For zeroForOne (token0 -> token1):
//   amountOut = liquidity * (sqrtPrice_old - sqrtPrice_new) / Q96
//   where sqrtPrice_new = liquidity * Q96 * sqrtPrice_old / (liquidity * Q96 + amountIn * sqrtPrice_old)
//
// For oneForZero (token1 -> token0):
//   amountOut = liquidity * Q96 * (1/sqrtPrice_old - 1/sqrtPrice_new)
func GetAmountOutV3(amountIn, sqrtPriceX96, liquidity *big.Int, zeroForOne bool, feeBps int) (*big.Int, error) {
	if amountIn == nil || sqrtPriceX96 == nil || liquidity == nil {
		return nil, fmt.Errorf("nil input to GetAmountOutV3")
	}
	if amountIn.Sign() <= 0 || sqrtPriceX96.Sign() <= 0 || liquidity.Sign() <= 0 {
		return nil, fmt.Errorf("invalid inputs")
	}

	// Apply fee
	feeMultiplier := big.NewInt(int64(10000 - feeBps))
	amountInAfterFee := MulDiv(amountIn, feeMultiplier, BPS)

	if zeroForOne {
		// token0 -> token1
		// newSqrtPrice = L * Q96 * oldSqrtPrice / (L * Q96 + amountIn * oldSqrtPrice)
		lq96 := new(big.Int).Mul(liquidity, Q96)
		numerator := new(big.Int).Mul(lq96, sqrtPriceX96)
		denominator := new(big.Int).Add(lq96, new(big.Int).Mul(amountInAfterFee, sqrtPriceX96))

		newSqrtPrice := new(big.Int).Div(numerator, denominator)

		// amountOut = L * (oldSqrtPrice - newSqrtPrice) / Q96
		diff := new(big.Int).Sub(sqrtPriceX96, newSqrtPrice)
		amountOut := MulDiv(liquidity, diff, Q96)
		return amountOut, nil
	}

	// token1 -> token0
	// newSqrtPrice = oldSqrtPrice + amountIn * Q96 / L
	delta := MulDiv(amountInAfterFee, Q96, liquidity)
	newSqrtPrice := new(big.Int).Add(sqrtPriceX96, delta)

	// amountOut = L * Q96 * (newSqrtPrice - oldSqrtPrice) / (oldSqrtPrice * newSqrtPrice)
	// = L * Q96 * delta / (old * new)
	lq96 := new(big.Int).Mul(liquidity, Q96)
	numerator := new(big.Int).Mul(lq96, delta)
	denominator := new(big.Int).Mul(sqrtPriceX96, newSqrtPrice)
	amountOut := new(big.Int).Div(numerator, denominator)

	return amountOut, nil
}

// GetAmountInV3 is the inverse of GetAmountOutV3 for single-tick swaps.
func GetAmountInV3(amountOut, sqrtPriceX96, liquidity *big.Int, zeroForOne bool, feeBps int) (*big.Int, error) {
	if amountOut == nil || sqrtPriceX96 == nil || liquidity == nil {
		return nil, fmt.Errorf("nil input to GetAmountInV3")
	}
	if amountOut.Sign() <= 0 || sqrtPriceX96.Sign() <= 0 || liquidity.Sign() <= 0 {
		return nil, fmt.Errorf("invalid inputs")
	}

	feeMultiplier := big.NewInt(int64(10000 - feeBps))

	if zeroForOne {
		// For token0 -> token1, we need to find amountIn that gives amountOut of token1
		// amountOut = L * (old - new) / Q96
		// new = old - amountOut * Q96 / L
		delta := MulDiv(amountOut, Q96, liquidity)
		newSqrtPrice := new(big.Int).Sub(sqrtPriceX96, delta)
		if newSqrtPrice.Sign() <= 0 {
			return nil, fmt.Errorf("price would go negative")
		}

		// amountIn = L * Q96 * (old - new) / (old * new) / feeMultiplier * BPS
		lq96 := new(big.Int).Mul(liquidity, Q96)
		numerator := new(big.Int).Mul(lq96, delta)
		denominator := new(big.Int).Mul(sqrtPriceX96, newSqrtPrice)
		amountInBeforeFee := new(big.Int).Div(numerator, denominator)

		amountIn := MulDivRoundUp(amountInBeforeFee, BPS, feeMultiplier)
		return amountIn, nil
	}

	// oneForZero: token1 -> token0
	// amountOut = L * Q96 * delta / (old * new)
	// Working backwards is complex; use estimation
	delta := MulDiv(amountOut, sqrtPriceX96, new(big.Int).Mul(liquidity, Q96))
	amountInBeforeFee := MulDiv(delta, liquidity, Q96)
	amountIn := MulDivRoundUp(amountInBeforeFee, BPS, feeMultiplier)
	return amountIn, nil
}

// SqrtPriceToPrice converts sqrtPriceX96 to a price ratio (token1/token0) scaled by 1e18.
func SqrtPriceToPrice(sqrtPriceX96 *big.Int) *big.Int {
	// price = (sqrtPriceX96 / 2^96)^2 = sqrtPriceX96^2 / 2^192
	sq := new(big.Int).Mul(sqrtPriceX96, sqrtPriceX96)
	// Scale by 1e18 before dividing
	sq.Mul(sq, Wei)
	q192 := new(big.Int).Lsh(One, 192)
	return new(big.Int).Div(sq, q192)
}
