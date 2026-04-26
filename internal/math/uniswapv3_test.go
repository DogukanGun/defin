package math

import (
	"math/big"
	"testing"
)

func e(n int64) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(n), nil)
}

func TestGetAmountOutV3_ZeroForOne(t *testing.T) {
	sqrtPriceX96 := new(big.Int).Mul(big.NewInt(4339505179), e(19)) // approximate

	liquidity := new(big.Int).Mul(e(15), big.NewInt(1000)) // 1e18

	amountIn := e(15) // 0.001 ETH

	out, err := GetAmountOutV3(amountIn, sqrtPriceX96, liquidity, true, 30)
	if err != nil {
		t.Fatal(err)
	}

	if out.Sign() <= 0 {
		t.Fatal("expected positive output")
	}

	t.Logf("V3 swap 0.001 ETH -> %s (with sqrtPriceX96=%s)", out.String(), sqrtPriceX96.String())
}

func TestGetAmountOutV3_OneForZero(t *testing.T) {
	sqrtPriceX96 := new(big.Int).Mul(big.NewInt(4339505179), e(19))
	liquidity := new(big.Int).Mul(e(15), big.NewInt(1000))
	amountIn := big.NewInt(3000000) // 3 USDC

	out, err := GetAmountOutV3(amountIn, sqrtPriceX96, liquidity, false, 30)
	if err != nil {
		t.Fatal(err)
	}

	if out.Sign() <= 0 {
		t.Fatal("expected positive output")
	}

	t.Logf("V3 swap 3 USDC -> %s wei ETH", out.String())
}

func TestGetAmountOutV3_ZeroInput(t *testing.T) {
	sqrtPriceX96 := e(30)
	liquidity := e(18)

	_, err := GetAmountOutV3(big.NewInt(0), sqrtPriceX96, liquidity, true, 30)
	if err == nil {
		t.Fatal("expected error for zero input")
	}
}

func TestSqrtPriceToPrice(t *testing.T) {
	sqrtPriceX96 := new(big.Int).Set(Q96) // price = 1.0

	price := SqrtPriceToPrice(sqrtPriceX96)

	expected := new(big.Int).Set(Wei)
	diff := new(big.Int).Sub(price, expected)
	diff.Abs(diff)

	tolerance := e(15) // 0.1% tolerance
	if diff.Cmp(tolerance) > 0 {
		t.Errorf("price=%s, expected ~%s", price.String(), expected.String())
	}
}

func BenchmarkGetAmountOutV3(b *testing.B) {
	sqrtPriceX96 := new(big.Int).Mul(big.NewInt(4339505179), e(19))
	liquidity := new(big.Int).Mul(e(15), big.NewInt(1000))
	amountIn := e(15)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetAmountOutV3(amountIn, sqrtPriceX96, liquidity, true, 30)
	}
}
