package math

import "math/big"

var (
	Zero = big.NewInt(0)
	One  = big.NewInt(1)
	Two  = big.NewInt(2)

	BPS       = big.NewInt(10000)
	Wei       = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil) // 1e18
	Q96       = new(big.Int).Exp(big.NewInt(2), big.NewInt(96), nil)  // 2^96
	MaxUint256 = new(big.Int).Sub(new(big.Int).Lsh(One, 256), One)
)

func MulDiv(a, b, denom *big.Int) *big.Int {
	if denom.Sign() == 0 {
		return new(big.Int)
	}
	result := new(big.Int).Mul(a, b)
	return result.Div(result, denom)
}

func MulDivRoundUp(a, b, denom *big.Int) *big.Int {
	if denom.Sign() == 0 {
		return new(big.Int)
	}
	result := new(big.Int).Mul(a, b)
	mod := new(big.Int).Mod(result, denom)
	result.Div(result, denom)
	if mod.Sign() > 0 {
		result.Add(result, One)
	}
	return result
}

func Sqrt(x *big.Int) *big.Int {
	if x.Sign() <= 0 {
		return new(big.Int)
	}
	z := new(big.Int).Set(x)
	y := new(big.Int).Add(new(big.Int).Rsh(z, 1), One)
	for y.Cmp(z) < 0 {
		z.Set(y)
		y.Add(new(big.Int).Div(x, z), z)
		y.Rsh(y, 1)
	}
	return z
}

func Min(a, b *big.Int) *big.Int {
	if a.Cmp(b) < 0 {
		return new(big.Int).Set(a)
	}
	return new(big.Int).Set(b)
}

func Max(a, b *big.Int) *big.Int {
	if a.Cmp(b) > 0 {
		return new(big.Int).Set(a)
	}
	return new(big.Int).Set(b)
}

func IsPositive(x *big.Int) bool {
	return x != nil && x.Sign() > 0
}
