package strategy

import (
	"context"
	"log/slog"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	mathutil "github.com/dogukangundogan/trader/internal/math"
	"github.com/dogukangundogan/trader/internal/monitor"
	"github.com/dogukangundogan/trader/internal/pool"
)

type Liquidation struct {
	enabled      bool
	minProfitWei *big.Int
	gasPrice     *big.Int
	healthMon    *monitor.HealthMonitor
	log          *slog.Logger
}

func NewLiquidation(enabled bool, minProfitWei, gasPrice *big.Int, healthMon *monitor.HealthMonitor, log *slog.Logger) *Liquidation {
	return &Liquidation{
		enabled:      enabled,
		minProfitWei: minProfitWei,
		gasPrice:     gasPrice,
		healthMon:    healthMon,
		log:          log,
	}
}

func (s *Liquidation) Name() string  { return "liquidation" }
func (s *Liquidation) Enabled() bool { return s.enabled }

func (s *Liquidation) Evaluate(ctx context.Context, registry *pool.Registry, blockNumber uint64) ([]Opportunity, error) {
	if s.healthMon == nil {
		return nil, nil
	}

	var opportunities []Opportunity

	// Get accounts below liquidation threshold
	accounts := s.healthMon.GetLiquidatable()

	for _, acct := range accounts {
		if acct.HealthFactor.Cmp(mathutil.Wei) >= 0 {
			continue // HF >= 1.0, not liquidatable
		}

		// Liquidate up to 50% of debt
		halfDebt := new(big.Int).Div(acct.DebtAmount, mathutil.Two)

		// Liquidation bonus is typically 5-15%
		// Gross profit = halfDebt * bonusBps / 10000
		bonusProfit := mathutil.MulDiv(halfDebt, big.NewInt(int64(acct.LiquidationBonus)), mathutil.BPS)

		gasEstimate := uint64(500000)
		netProfit := mathutil.NetProfit(bonusProfit, big.NewInt(int64(gasEstimate)), s.gasPrice, halfDebt)

		if !mathutil.IsProfitable(netProfit, s.minProfitWei) {
			continue
		}

		s.log.Info("liquidation opportunity found",
			"account", acct.Address.Hex(),
			"health_factor", acct.HealthFactor.String(),
			"debt_token", acct.DebtToken.Hex(),
			"collateral_token", acct.CollateralToken.Hex(),
			"half_debt", halfDebt.String(),
			"net_profit", netProfit.String(),
			"block", blockNumber,
		)

		opportunities = append(opportunities, Opportunity{
			StrategyName: s.Name(),
			ChainID:      acct.ChainID,
			GrossProfit:  bonusProfit,
			NetProfit:    netProfit,
			GasEstimate:  gasEstimate,
			Steps: []SwapStep{
				{
					TokenIn:  acct.DebtToken,
					TokenOut: acct.CollateralToken,
					AmountIn: halfDebt,
				},
			},
			FlashLoan: &FlashLoanParams{
				Token:    acct.DebtToken,
				Amount:   halfDebt,
				Provider: "aave_v3",
			},
		})
	}

	return opportunities, nil
}

// Unused but satisfies interface for Pool field in SwapStep
var _ common.Address
