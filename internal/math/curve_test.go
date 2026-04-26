package math

import (
	"math/big"
	"testing"
)

func TestGetD(t *testing.T) {
	// 2-coin pool with equal balances
	balances := []*big.Int{
		new(big.Int).Mul(big.NewInt(1000000), big.NewInt(1e18)),
		new(big.Int).Mul(big.NewInt(1000000), big.NewInt(1e18)),
	}
	amp := big.NewInt(100)

	d, err := GetD(balances, amp, 2)
	if err != nil {
		t.Fatal(err)
	}

	// D should be close to 2 * 1e6 * 1e18 = 2e24
	expected := new(big.Int).Mul(big.NewInt(2000000), big.NewInt(1e18))
	diff := new(big.Int).Sub(d, expected)
	diff.Abs(diff)

	// Allow 0.01% tolerance
	tolerance := new(big.Int).Div(expected, big.NewInt(10000))
	if diff.Cmp(tolerance) > 0 {
		t.Errorf("D=%s, expected ~%s (diff=%s)", d.String(), expected.String(), diff.String())
	}
}

func TestGetD_Imbalanced(t *testing.T) {
	// Imbalanced pool
	balances := []*big.Int{
		new(big.Int).Mul(big.NewInt(900000), big.NewInt(1e18)),
		new(big.Int).Mul(big.NewInt(1100000), big.NewInt(1e18)),
	}
	amp := big.NewInt(100)

	d, err := GetD(balances, amp, 2)
	if err != nil {
		t.Fatal(err)
	}

	if d.Sign() <= 0 {
		t.Fatal("D should be positive")
	}

	t.Logf("D for imbalanced pool: %s", d.String())
}

func TestGetDyCurve(t *testing.T) {
	balances := []*big.Int{
		new(big.Int).Mul(big.NewInt(1000000), big.NewInt(1e18)), // 1M token0
		new(big.Int).Mul(big.NewInt(1000000), big.NewInt(1e18)), // 1M token1
	}
	amp := big.NewInt(100)
	amountIn := new(big.Int).Mul(big.NewInt(1000), big.NewInt(1e18)) // swap 1000 tokens

	dy, err := GetDyCurve(balances, amp, amountIn, 0, 1, 2, 4) // 4 bps fee
	if err != nil {
		t.Fatal(err)
	}

	if dy.Sign() <= 0 {
		t.Fatal("expected positive dy")
	}

	// For a StableSwap with balanced pool and high A, output should be close to input
	// (minus fee and small price impact)
	ratio := new(big.Int).Mul(dy, big.NewInt(10000))
	ratio.Div(ratio, amountIn)
	// Should be 9990+ (99.9%+ of input)
	if ratio.Int64() < 9900 {
		t.Errorf("output ratio too low: %d/10000, dy=%s", ratio.Int64(), dy.String())
	}

	t.Logf("input=%s, output=%s, ratio=%d/10000", amountIn.String(), dy.String(), ratio.Int64())
}

func TestGetDyCurve_Imbalanced(t *testing.T) {
	// Imbalanced: swapping into the larger side should give less
	balances := []*big.Int{
		new(big.Int).Mul(big.NewInt(800000), big.NewInt(1e18)),  // 800K token0
		new(big.Int).Mul(big.NewInt(1200000), big.NewInt(1e18)), // 1.2M token1
	}
	amp := big.NewInt(100)
	amountIn := new(big.Int).Mul(big.NewInt(10000), big.NewInt(1e18))

	// Swap token0 -> token1 (adding to smaller side, getting from larger)
	dy01, err := GetDyCurve(balances, amp, amountIn, 0, 1, 2, 4)
	if err != nil {
		t.Fatal(err)
	}

	// Swap token1 -> token0 (adding to larger side, getting from smaller)
	dy10, err := GetDyCurve(balances, amp, amountIn, 1, 0, 2, 4)
	if err != nil {
		t.Fatal(err)
	}

	// Swapping 0->1 should give more output (buying from the excess side)
	if dy01.Cmp(dy10) <= 0 {
		t.Errorf("expected dy01 > dy10 for imbalanced pool, got dy01=%s, dy10=%s",
			dy01.String(), dy10.String())
	}

	t.Logf("0->1: %s, 1->0: %s", dy01.String(), dy10.String())
}

func BenchmarkGetD(b *testing.B) {
	balances := []*big.Int{
		new(big.Int).Mul(big.NewInt(1000000), big.NewInt(1e18)),
		new(big.Int).Mul(big.NewInt(1000000), big.NewInt(1e18)),
	}
	amp := big.NewInt(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetD(balances, amp, 2)
	}
}

func BenchmarkGetDyCurve(b *testing.B) {
	balances := []*big.Int{
		new(big.Int).Mul(big.NewInt(1000000), big.NewInt(1e18)),
		new(big.Int).Mul(big.NewInt(1000000), big.NewInt(1e18)),
	}
	amp := big.NewInt(100)
	amountIn := new(big.Int).Mul(big.NewInt(1000), big.NewInt(1e18))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetDyCurve(balances, amp, amountIn, 0, 1, 2, 4)
	}
}
