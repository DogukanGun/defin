package math

import (
	"math/big"
	"testing"
)

func TestSqrt(t *testing.T) {
	tests := []struct {
		input    int64
		expected int64
	}{
		{0, 0},
		{1, 1},
		{4, 2},
		{9, 3},
		{100, 10},
		{101, 10}, // floor
		{1000000, 1000},
	}

	for _, tt := range tests {
		result := Sqrt(big.NewInt(tt.input))
		if result.Int64() != tt.expected {
			t.Errorf("Sqrt(%d) = %d, want %d", tt.input, result.Int64(), tt.expected)
		}
	}
}

func TestSqrt_Large(t *testing.T) {
	// sqrt(1e36) = 1e18
	input := new(big.Int).Exp(big.NewInt(10), big.NewInt(36), nil)
	expected := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)

	result := Sqrt(input)
	if result.Cmp(expected) != 0 {
		t.Errorf("Sqrt(1e36) = %s, want %s", result.String(), expected.String())
	}
}

func TestMulDiv(t *testing.T) {
	a := big.NewInt(1000)
	b := big.NewInt(3000)
	denom := big.NewInt(10000)

	result := MulDiv(a, b, denom)
	if result.Int64() != 300 {
		t.Errorf("MulDiv(1000, 3000, 10000) = %d, want 300", result.Int64())
	}
}

func TestMulDivRoundUp(t *testing.T) {
	a := big.NewInt(10)
	b := big.NewInt(3)
	denom := big.NewInt(10)

	result := MulDivRoundUp(a, b, denom)
	// 10*3/10 = 3, no remainder
	if result.Int64() != 3 {
		t.Errorf("expected 3, got %d", result.Int64())
	}

	// 10*3/7 = 4.28 -> 5
	result = MulDivRoundUp(big.NewInt(10), big.NewInt(3), big.NewInt(7))
	if result.Int64() != 5 {
		t.Errorf("expected 5, got %d", result.Int64())
	}
}

func TestMinMax(t *testing.T) {
	a := big.NewInt(5)
	b := big.NewInt(10)

	if Min(a, b).Int64() != 5 {
		t.Error("Min failed")
	}
	if Max(a, b).Int64() != 10 {
		t.Error("Max failed")
	}
}

func BenchmarkSqrt(b *testing.B) {
	input := new(big.Int).Exp(big.NewInt(10), big.NewInt(36), nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Sqrt(input)
	}
}
