package math

import "math/big"

// GetAmountOutV2 calculates the output amount for a Uniswap V2 swap.
// Formula: amountOut = (amountIn * feeMultiplier * reserveOut) / (reserveIn * 10000 + amountIn * feeMultiplier)
// where feeMultiplier = 10000 - feeBps
func GetAmountOutV2(amountIn, reserveIn, reserveOut *big.Int, feeBps int) *big.Int {
	if amountIn.Sign() <= 0 || reserveIn.Sign() <= 0 || reserveOut.Sign() <= 0 {
		return new(big.Int)
	}

	feeMultiplier := big.NewInt(int64(10000 - feeBps))

	amountInWithFee := new(big.Int).Mul(amountIn, feeMultiplier)
	numerator := new(big.Int).Mul(amountInWithFee, reserveOut)

	denominator := new(big.Int).Mul(reserveIn, BPS)
	denominator.Add(denominator, amountInWithFee)

	return new(big.Int).Div(numerator, denominator)
}

// GetAmountInV2 calculates the required input amount for a desired output.
// Formula: amountIn = (reserveIn * amountOut * 10000) / ((reserveOut - amountOut) * feeMultiplier) + 1
func GetAmountInV2(amountOut, reserveIn, reserveOut *big.Int, feeBps int) *big.Int {
	if amountOut.Sign() <= 0 || reserveIn.Sign() <= 0 || reserveOut.Sign() <= 0 {
		return new(big.Int)
	}
	if amountOut.Cmp(reserveOut) >= 0 {
		return new(big.Int) // cannot withdraw more than reserve
	}

	feeMultiplier := big.NewInt(int64(10000 - feeBps))

	numerator := new(big.Int).Mul(reserveIn, amountOut)
	numerator.Mul(numerator, BPS)

	denominator := new(big.Int).Sub(reserveOut, amountOut)
	denominator.Mul(denominator, feeMultiplier)

	result := new(big.Int).Div(numerator, denominator)
	result.Add(result, One)

	return result
}

// OptimalArbInputV2 finds the optimal input amount for cross-DEX arbitrage
// between two constant product pools using ternary search.
// Pool A: buy (send token0, get token1). reserveIn=r0a, reserveOut=r1a
// Pool B: sell (send token1, get token0). reserveIn=r0b, reserveOut=r1b
func OptimalArbInputV2(r0a, r1a *big.Int, feeBpsA int, r0b, r1b *big.Int, feeBpsB int) *big.Int {
	// Quick check: is there any arb at all?
	// Marginal rate check: buying epsilon on A and selling on B must be profitable.
	// On A: 1 unit of token0 buys ~(r1a * fmA) / (r0a * 10000) of token1
	// On B: 1 unit of token1 buys ~(r1b * fmB) / (r0b * 10000) of token0
	// Profitable if product of rates > 1: r1a*fmA*r1b*fmB > r0a*r0b*10000^2
	fmA := big.NewInt(int64(10000 - feeBpsA))
	fmB := big.NewInt(int64(10000 - feeBpsB))

	lhs := new(big.Int).Mul(r1a, fmA)
	lhs.Mul(lhs, r1b)
	lhs.Mul(lhs, fmB)

	rhs := new(big.Int).Mul(r0a, r0b)
	rhs.Mul(rhs, BPS)
	rhs.Mul(rhs, BPS)

	if lhs.Cmp(rhs) <= 0 {
		return new(big.Int) // No arb
	}

	// Ternary search over the unimodal profit function
	low := big.NewInt(1)
	upper := Min(r0a, r0b)
	high := new(big.Int).Rsh(upper, 1)

	for i := 0; i < 200; i++ {
		diff := new(big.Int).Sub(high, low)
		if diff.Cmp(big.NewInt(2)) <= 0 {
			break
		}
		third := new(big.Int).Div(diff, big.NewInt(3))

		m1 := new(big.Int).Add(low, third)
		m2 := new(big.Int).Sub(high, third)

		p1 := ProfitV2(m1, r0a, r1a, feeBpsA, r0b, r1b, feeBpsB)
		p2 := ProfitV2(m2, r0a, r1a, feeBpsA, r0b, r1b, feeBpsB)

		if p1.Cmp(p2) < 0 {
			low = m1
		} else {
			high = m2
		}
	}

	best := new(big.Int).Add(low, high)
	best.Rsh(best, 1)

	profit := ProfitV2(best, r0a, r1a, feeBpsA, r0b, r1b, feeBpsB)
	if profit.Sign() <= 0 {
		return new(big.Int)
	}

	return best
}

// ProfitV2 calculates the profit from buying on pool A and selling on pool B.
func ProfitV2(amountIn, r0a, r1a *big.Int, feeBpsA int, r0b, r1b *big.Int, feeBpsB int) *big.Int {
	// Buy token1 on pool A (token0 -> token1)
	amountMid := GetAmountOutV2(amountIn, r0a, r1a, feeBpsA)
	if amountMid.Sign() <= 0 {
		return new(big.Int)
	}

	// Sell token1 on pool B (token1 -> token0)
	amountOut := GetAmountOutV2(amountMid, r0b, r1b, feeBpsB)

	profit := new(big.Int).Sub(amountOut, amountIn)
	return profit
}
