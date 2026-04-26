package strategy

import (
	"context"
	"log/slog"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	mathutil "github.com/dogukangundogan/trader/internal/math"
	"github.com/dogukangundogan/trader/internal/pool"
)

type CrossDex struct {
	enabled      bool
	minProfitWei *big.Int
	gasPrice     *big.Int
	log          *slog.Logger
}

func NewCrossDex(enabled bool, minProfitWei, gasPrice *big.Int, log *slog.Logger) *CrossDex {
	return &CrossDex{
		enabled:      enabled,
		minProfitWei: minProfitWei,
		gasPrice:     gasPrice,
		log:          log,
	}
}

func (s *CrossDex) Name() string  { return "cross_dex" }
func (s *CrossDex) Enabled() bool { return s.enabled }

func (s *CrossDex) Evaluate(ctx context.Context, registry *pool.Registry, blockNumber uint64) ([]Opportunity, error) {
	var opportunities []Opportunity

	allPools := registry.All()

	// Group pools by canonical (token0, token1, chainID) — works for V2 and V3
	type pairKey struct {
		token0  common.Address
		token1  common.Address
		chainID int64
	}
	pairPools := make(map[pairKey][]pool.Pool)

	for _, p := range allPools {
		if p.State() == nil {
			continue
		}
		t0, t1 := p.Token0(), p.Token1()
		if t0.Hex() > t1.Hex() {
			t0, t1 = t1, t0
		}
		key := pairKey{token0: t0, token1: t1, chainID: p.ChainID()}
		pairPools[key] = append(pairPools[key], p)
	}

	for key, pools := range pairPools {
		if len(pools) < 2 {
			continue
		}
		for i := 0; i < len(pools); i++ {
			for j := i + 1; j < len(pools); j++ {
				opps := s.findArb(pools[i], pools[j], key.token0, key.token1, blockNumber)
				opportunities = append(opportunities, opps...)
			}
		}
	}

	return opportunities, nil
}

func (s *CrossDex) findArb(poolA, poolB pool.Pool, token0, token1 common.Address, blockNumber uint64) []Opportunity {
	var opps []Opportunity

	if opp := s.checkDirection(poolA, poolB, token0, token1, blockNumber); opp != nil {
		opps = append(opps, *opp)
	}
	if opp := s.checkDirection(poolB, poolA, token0, token1, blockNumber); opp != nil {
		opps = append(opps, *opp)
	}

	return opps
}

func (s *CrossDex) checkDirection(buyPool, sellPool pool.Pool, token0, token1 common.Address, blockNumber uint64) *Opportunity {
	profitFn := func(amtIn *big.Int) *big.Int {
		mid, err := buyPool.GetAmountOut(token0, amtIn)
		if err != nil || mid.Sign() <= 0 {
			return big.NewInt(-1)
		}
		out, err := sellPool.GetAmountOut(token1, mid)
		if err != nil || out.Sign() <= 0 {
			return big.NewInt(-1)
		}
		return new(big.Int).Sub(out, amtIn)
	}

	low := new(big.Int).Exp(big.NewInt(10), big.NewInt(15), nil)
	high := new(big.Int).Exp(big.NewInt(10), big.NewInt(21), nil)

	var bestProfit, bestInput *big.Int
	for iter := 0; iter < 64; iter++ {
		mid := new(big.Int).Rsh(new(big.Int).Add(low, high), 1)
		p := profitFn(mid)
		if p.Sign() > 0 {
			if bestProfit == nil || p.Cmp(bestProfit) > 0 {
				bestProfit = new(big.Int).Set(p)
				bestInput = new(big.Int).Set(mid)
			}
			midUp := new(big.Int).Add(mid, new(big.Int).Rsh(new(big.Int).Sub(high, mid), 1))
			if profitFn(midUp).Cmp(p) > 0 {
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

	gasEstimate := uint64(250000)
	netProfit := mathutil.NetProfit(bestProfit, big.NewInt(int64(gasEstimate)), s.gasPrice, bestInput)

	if !mathutil.IsProfitable(netProfit, s.minProfitWei) {
		return nil
	}

	s.log.Info("cross-dex opportunity found",
		"buy_pool", buyPool.Address().Hex(),
		"sell_pool", sellPool.Address().Hex(),
		"buy_type", string(buyPool.Type()),
		"sell_type", string(sellPool.Type()),
		"optimal_in", bestInput.String(),
		"gross_profit", bestProfit.String(),
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
			{Pool: buyPool, TokenIn: token0, TokenOut: token1, AmountIn: bestInput},
			{Pool: sellPool, TokenIn: token1, TokenOut: token0},
		},
		FlashLoan: &FlashLoanParams{
			Token:    token0,
			Amount:   bestInput,
			Provider: "aave_v3",
		},
	}
}
