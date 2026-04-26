package strategy

import (
	"context"
	"log/slog"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	mathutil "github.com/dogukangundogan/trader/internal/math"
	"github.com/dogukangundogan/trader/internal/pool"
)

type CurveStable struct {
	enabled      bool
	minProfitWei *big.Int
	gasPrice     *big.Int
	log          *slog.Logger
}

func NewCurveStable(enabled bool, minProfitWei, gasPrice *big.Int, log *slog.Logger) *CurveStable {
	return &CurveStable{
		enabled:      enabled,
		minProfitWei: minProfitWei,
		gasPrice:     gasPrice,
		log:          log,
	}
}

func (s *CurveStable) Name() string  { return "curve_stable" }
func (s *CurveStable) Enabled() bool { return s.enabled }

func (s *CurveStable) Evaluate(ctx context.Context, registry *pool.Registry, blockNumber uint64) ([]Opportunity, error) {
	var opportunities []Opportunity

	allPools := registry.All()

	// Find Curve pools and matching V2 pools for the same pair
	var curvePools []pool.Pool
	for _, p := range allPools {
		if p.Type() == pool.TypeCurve && p.State() != nil {
			curvePools = append(curvePools, p)
		}
	}

	for _, cp := range curvePools {
		t0, t1 := cp.Token0(), cp.Token1()

		// Find V2 pools with the same pair
		v2Pools := registry.GetByPair(t0, t1, cp.ChainID())
		for _, v2 := range v2Pools {
			if v2.Type() != pool.TypeUniswapV2 || v2.State() == nil {
				continue
			}

			// Check both directions: curve->v2 and v2->curve
			if opp := s.checkCurveToV2(cp, v2, t0, t1, blockNumber); opp != nil {
				opportunities = append(opportunities, *opp)
			}
			if opp := s.checkV2ToCurve(v2, cp, t0, t1, blockNumber); opp != nil {
				opportunities = append(opportunities, *opp)
			}
		}
	}

	return opportunities, nil
}

func (s *CurveStable) checkCurveToV2(curvePool, v2Pool pool.Pool, token0, token1 common.Address, blockNumber uint64) *Opportunity {
	// Buy on Curve (token0 -> token1), sell on V2 (token1 -> token0)
	return s.checkDirection(curvePool, v2Pool, token0, token1, blockNumber)
}

func (s *CurveStable) checkV2ToCurve(v2Pool, curvePool pool.Pool, token0, token1 common.Address, blockNumber uint64) *Opportunity {
	// Buy on V2 (token0 -> token1), sell on Curve (token1 -> token0)
	return s.checkDirection(v2Pool, curvePool, token0, token1, blockNumber)
}

func (s *CurveStable) checkDirection(buyPool, sellPool pool.Pool, tokenIn, tokenOut common.Address, blockNumber uint64) *Opportunity {
	// Binary search for optimal input
	low := new(big.Int).Exp(big.NewInt(10), big.NewInt(15), nil)
	high := new(big.Int).Exp(big.NewInt(10), big.NewInt(23), nil)

	var bestProfit *big.Int
	var bestInput *big.Int

	for iter := 0; iter < 64; iter++ {
		mid := new(big.Int).Add(low, high)
		mid.Rsh(mid, 1)

		profit := s.simulate(mid, buyPool, sellPool, tokenIn, tokenOut)
		if profit.Sign() > 0 {
			if bestProfit == nil || profit.Cmp(bestProfit) > 0 {
				bestProfit = new(big.Int).Set(profit)
				bestInput = new(big.Int).Set(mid)
			}
			midUp := new(big.Int).Add(mid, new(big.Int).Rsh(new(big.Int).Sub(high, mid), 1))
			profitUp := s.simulate(midUp, buyPool, sellPool, tokenIn, tokenOut)
			if profitUp.Cmp(profit) > 0 {
				low = mid
			} else {
				high = mid
			}
		} else {
			high = mid
		}
	}

	if bestProfit == nil || bestProfit.Sign() <= 0 {
		return nil
	}

	gasEstimate := uint64(350000)
	netProfit := mathutil.NetProfit(bestProfit, big.NewInt(int64(gasEstimate)), s.gasPrice, bestInput)

	if !mathutil.IsProfitable(netProfit, s.minProfitWei) {
		return nil
	}

	s.log.Info("curve-stable opportunity found",
		"buy_pool", buyPool.Address().Hex(),
		"sell_pool", sellPool.Address().Hex(),
		"input", bestInput.String(),
		"net_profit", netProfit.String(),
		"block", blockNumber,
	)

	return &Opportunity{
		StrategyName: s.Name(),
		ChainID:      buyPool.ChainID(),
		GrossProfit:  bestProfit,
		NetProfit:    netProfit,
		GasEstimate:  gasEstimate,
		Steps: []SwapStep{
			{Pool: buyPool, TokenIn: tokenIn, TokenOut: tokenOut, AmountIn: bestInput},
			{Pool: sellPool, TokenIn: tokenOut, TokenOut: tokenIn},
		},
		FlashLoan: &FlashLoanParams{
			Token:    tokenIn,
			Amount:   bestInput,
			Provider: "aave_v3",
		},
	}
}

func (s *CurveStable) simulate(amountIn *big.Int, buyPool, sellPool pool.Pool, tokenIn, tokenOut common.Address) *big.Int {
	out1, err := buyPool.GetAmountOut(tokenIn, amountIn)
	if err != nil || out1.Sign() <= 0 {
		return big.NewInt(0)
	}
	out2, err := sellPool.GetAmountOut(tokenOut, out1)
	if err != nil || out2.Sign() <= 0 {
		return big.NewInt(0)
	}
	return new(big.Int).Sub(out2, amountIn)
}
