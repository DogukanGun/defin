package math

import "math/big"

// NetProfit computes grossProfit - gasCost - flashLoanFee.
// gasCost = gasUsed * gasPrice
// flashLoanFee = principal * 5 / 10000 (Aave v3 = 0.05%)
func NetProfit(grossProfit, gasUsed, gasPrice *big.Int, flashLoanPrincipal *big.Int) *big.Int {
	gasCost := new(big.Int).Mul(gasUsed, gasPrice)

	net := new(big.Int).Sub(grossProfit, gasCost)

	if flashLoanPrincipal != nil && flashLoanPrincipal.Sign() > 0 {
		// Flash loan fee: 0.05% = 5 bps
		fee := MulDiv(flashLoanPrincipal, big.NewInt(5), BPS)
		net.Sub(net, fee)
	}

	return net
}

// IsProfitable returns true if net profit exceeds minProfitWei.
func IsProfitable(netProfit, minProfitWei *big.Int) bool {
	return netProfit.Cmp(minProfitWei) > 0
}
