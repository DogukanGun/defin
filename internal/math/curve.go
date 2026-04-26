package math

import (
	"fmt"
	"math/big"
)

// GetD computes the StableSwap invariant D using Newton-Raphson iteration.
//
// StableSwap invariant:
//   A * n^n * sum(x_i) + D = A * D * n^n + D^(n+1) / (n^n * prod(x_i))
//
// Newton-Raphson:
//   D_new = (A * n * S + n * D_P) * D / ((A * n - 1) * D + (n + 1) * D_P)
//   where S = sum(x_i), D_P = D^(n+1) / (n^n * prod(x_i))
func GetD(balances []*big.Int, amp *big.Int, nCoins int) (*big.Int, error) {
	n := big.NewInt(int64(nCoins))

	// S = sum of balances
	s := new(big.Int)
	for _, b := range balances {
		s.Add(s, b)
	}

	if s.Sign() == 0 {
		return new(big.Int), nil
	}

	d := new(big.Int).Set(s)
	ann := new(big.Int).Mul(amp, n) // A * n

	for i := 0; i < 256; i++ {
		dP := new(big.Int).Set(d)
		for _, x := range balances {
			// dP = dP * D / (x * n)
			dP.Mul(dP, d)
			xn := new(big.Int).Mul(x, n)
			if xn.Sign() == 0 {
				return nil, fmt.Errorf("zero balance in pool")
			}
			dP.Div(dP, xn)
		}

		prevD := new(big.Int).Set(d)

		// numerator = (A*n*S + n*D_P) * D
		annS := new(big.Int).Mul(ann, s)
		nDp := new(big.Int).Mul(n, dP)
		num := new(big.Int).Add(annS, nDp)
		num.Mul(num, d)

		// denominator = (A*n - 1) * D + (n+1) * D_P
		annMinus1 := new(big.Int).Sub(ann, One)
		nPlus1 := new(big.Int).Add(n, One)
		denom := new(big.Int).Mul(annMinus1, d)
		denom.Add(denom, new(big.Int).Mul(nPlus1, dP))

		d.Div(num, denom)

		// Check convergence (|d - prevD| <= 1)
		diff := new(big.Int).Sub(d, prevD)
		diff.Abs(diff)
		if diff.Cmp(One) <= 0 {
			return d, nil
		}
	}

	return nil, fmt.Errorf("GetD did not converge")
}

// GetY computes the new balance of token j after swapping, using Newton-Raphson.
// Given D and new balance of token i, find y = balance[j].
func GetY(balances []*big.Int, amp *big.Int, i, j, nCoins int, newXi *big.Int) (*big.Int, error) {
	n := big.NewInt(int64(nCoins))

	d, err := GetD(balances, amp, nCoins)
	if err != nil {
		return nil, err
	}

	ann := new(big.Int).Mul(amp, n)

	// c = D^(n+1) / (n^n * prod(x_k for k != j) * A * n)
	c := new(big.Int).Set(d)
	s := new(big.Int)

	for k := 0; k < nCoins; k++ {
		var xk *big.Int
		if k == i {
			xk = newXi
		} else if k == j {
			continue
		} else {
			xk = balances[k]
		}
		s.Add(s, xk)
		c.Mul(c, d)
		c.Div(c, new(big.Int).Mul(xk, n))
	}

	c.Mul(c, d)
	c.Div(c, new(big.Int).Mul(ann, n))

	b := new(big.Int).Add(s, new(big.Int).Div(d, ann))

	y := new(big.Int).Set(d)
	for iter := 0; iter < 256; iter++ {
		prevY := new(big.Int).Set(y)

		// y_new = (y^2 + c) / (2*y + b - D)
		ySquared := new(big.Int).Mul(y, y)
		num := new(big.Int).Add(ySquared, c)
		denom := new(big.Int).Add(new(big.Int).Mul(Two, y), b)
		denom.Sub(denom, d)

		y.Div(num, denom)

		diff := new(big.Int).Sub(y, prevY)
		diff.Abs(diff)
		if diff.Cmp(One) <= 0 {
			return y, nil
		}
	}

	return nil, fmt.Errorf("GetY did not converge")
}

// GetDyCurve calculates the output amount for a Curve StableSwap.
// dy = balances[j] - GetY(newXi) - 1 (apply fee after)
func GetDyCurve(balances []*big.Int, amp, amountIn *big.Int, i, j, nCoins, feeBps int) (*big.Int, error) {
	newXi := new(big.Int).Add(balances[i], amountIn)

	newYj, err := GetY(balances, amp, i, j, nCoins, newXi)
	if err != nil {
		return nil, err
	}

	dy := new(big.Int).Sub(balances[j], newYj)
	dy.Sub(dy, One) // rounding

	if dy.Sign() <= 0 {
		return new(big.Int), nil
	}

	// Apply fee
	fee := MulDiv(dy, big.NewInt(int64(feeBps)), BPS)
	dy.Sub(dy, fee)

	return dy, nil
}
