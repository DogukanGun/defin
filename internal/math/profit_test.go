package math

import (
	"math/big"
	"testing"
)

func TestNetProfit(t *testing.T) {
	grossProfit := big.NewInt(1e16)  // 0.01 ETH
	gasUsed := big.NewInt(250000)
	gasPrice := big.NewInt(1e9)      // 1 gwei
	flashLoanPrincipal := big.NewInt(1e18) // 1 ETH

	net := NetProfit(grossProfit, gasUsed, gasPrice, flashLoanPrincipal)

	// gross = 0.01 ETH = 1e16
	// gas = 250000 * 1e9 = 2.5e14
	// flash fee = 1e18 * 5 / 10000 = 5e14
	// net = 1e16 - 2.5e14 - 5e14 = 9.25e15
	expected := big.NewInt(9250000000000000)
	if net.Cmp(expected) != 0 {
		t.Errorf("net profit = %s, want %s", net.String(), expected.String())
	}
}

func TestNetProfit_NoFlashLoan(t *testing.T) {
	grossProfit := big.NewInt(1e16)
	gasUsed := big.NewInt(200000)
	gasPrice := big.NewInt(1e9)

	net := NetProfit(grossProfit, gasUsed, gasPrice, nil)

	expected := big.NewInt(1e16 - 200000*1e9)
	if net.Cmp(expected) != 0 {
		t.Errorf("net profit = %s, want %s", net.String(), expected.String())
	}
}

func TestIsProfitable(t *testing.T) {
	if IsProfitable(big.NewInt(100), big.NewInt(200)) {
		t.Error("100 should not be profitable with min 200")
	}
	if !IsProfitable(big.NewInt(300), big.NewInt(200)) {
		t.Error("300 should be profitable with min 200")
	}
}
