package strategy

import (
	"context"
	"log/slog"
	"math/big"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common"

	mathutil "github.com/dogukangundogan/trader/internal/math"
	"github.com/dogukangundogan/trader/internal/pool"
)

type Triangular struct {
	enabled      bool
	minProfitWei *big.Int
	maxHops      int
	gasPrice     *big.Int
	log          *slog.Logger
}

func NewTriangular(enabled bool, minProfitWei, gasPrice *big.Int, maxHops int, log *slog.Logger) *Triangular {
	if maxHops == 0 {
		maxHops = 3
	}
	return &Triangular{
		enabled:      enabled,
		minProfitWei: minProfitWei,
		maxHops:      maxHops,
		gasPrice:     gasPrice,
		log:          log,
	}
}

func (s *Triangular) Name() string  { return "triangular" }
func (s *Triangular) Enabled() bool { return s.enabled }

type triEdge struct {
	pool     pool.Pool
	tokenOut common.Address
}

func (s *Triangular) Evaluate(ctx context.Context, registry *pool.Registry, blockNumber uint64) ([]Opportunity, error) {
	var opportunities []Opportunity

	allPools := registry.All()

	graph := make(map[common.Address][]triEdge)
	for _, p := range allPools {
		if p.State() == nil {
			continue
		}
		t0, t1 := p.Token0(), p.Token1()
		graph[t0] = append(graph[t0], triEdge{pool: p, tokenOut: t1})
		graph[t1] = append(graph[t1], triEdge{pool: p, tokenOut: t0})
	}

	visited := make(map[string]bool)
	for startToken := range graph {
		s.dfs(graph, startToken, startToken, nil, visited, blockNumber, &opportunities)
	}

	return opportunities, nil
}

func (s *Triangular) dfs(
	graph map[common.Address][]triEdge,
	startToken, currentToken common.Address,
	path []triEdge,
	visited map[string]bool,
	blockNumber uint64,
	opps *[]Opportunity,
) {
	if len(path) >= s.maxHops {
		return
	}
	for _, e := range graph[currentToken] {
		if e.tokenOut == startToken {
			if len(path) < 2 {
				continue // 2-hop is cross-dex territory
			}
			fullPath := append(path, e)
			key := cycleKey(fullPath)
			if !visited[key] {
				visited[key] = true
				if opp := s.evaluateCycle(startToken, fullPath, blockNumber); opp != nil {
					*opps = append(*opps, *opp)
				}
			}
			continue
		}

		// Avoid using the same pool twice in a path
		duplicate := false
		for _, pe := range path {
			if pe.pool.Address() == e.pool.Address() {
				duplicate = true
				break
			}
		}
		if duplicate {
			continue
		}

		s.dfs(graph, startToken, e.tokenOut, append(path, e), visited, blockNumber, opps)
	}
}

func (s *Triangular) evaluateCycle(startToken common.Address, path []triEdge, blockNumber uint64) *Opportunity {
	low := new(big.Int).Exp(big.NewInt(10), big.NewInt(15), nil)
	high := new(big.Int).Exp(big.NewInt(10), big.NewInt(21), nil)

	var bestProfit *big.Int
	var bestInput *big.Int

	for iter := 0; iter < 64; iter++ {
		mid := new(big.Int).Add(low, high)
		mid.Rsh(mid, 1)

		profit := s.simulateCycle(mid, startToken, path)
		if profit.Sign() > 0 {
			if bestProfit == nil || profit.Cmp(bestProfit) > 0 {
				bestProfit = new(big.Int).Set(profit)
				bestInput = new(big.Int).Set(mid)
			}

			midUp := new(big.Int).Add(mid, new(big.Int).Rsh(new(big.Int).Sub(high, mid), 1))
			if s.simulateCycle(midUp, startToken, path).Cmp(profit) > 0 {
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

	gasEstimate := uint64(200000 + 150000*uint64(len(path)))
	netProfit := mathutil.NetProfit(bestProfit, big.NewInt(int64(gasEstimate)), s.gasPrice, bestInput)

	if !mathutil.IsProfitable(netProfit, s.minProfitWei) {
		return nil
	}

	poolAddrs := make([]string, len(path))
	for i, e := range path {
		poolAddrs[i] = e.pool.Address().Hex()
	}

	s.log.Info("triangular opportunity found",
		"start_token", startToken.Hex(),
		"pools", poolAddrs,
		"hops", len(path),
		"input", bestInput.String(),
		"gross_profit", bestProfit.String(),
		"net_profit", netProfit.String(),
		"block", blockNumber,
	)

	steps := make([]SwapStep, len(path))
	tokenIn := startToken
	for i, e := range path {
		amtIn := new(big.Int)
		if i == 0 {
			amtIn.Set(bestInput)
		}
		steps[i] = SwapStep{Pool: e.pool, TokenIn: tokenIn, TokenOut: e.tokenOut, AmountIn: amtIn}
		tokenIn = e.tokenOut
	}

	return &Opportunity{
		StrategyName: s.Name(),
		ChainID:      path[0].pool.ChainID(),
		GrossProfit:  bestProfit,
		NetProfit:    netProfit,
		GasEstimate:  gasEstimate,
		Steps:        steps,
		FlashLoan: &FlashLoanParams{
			Token:    startToken,
			Amount:   bestInput,
			Provider: "aave_v3",
		},
	}
}

func (s *Triangular) simulateCycle(amountIn *big.Int, startToken common.Address, path []triEdge) *big.Int {
	current := startToken
	amount := new(big.Int).Set(amountIn)

	for _, e := range path {
		out, err := e.pool.GetAmountOut(current, amount)
		if err != nil || out.Sign() <= 0 {
			return big.NewInt(0)
		}
		amount = out
		current = e.tokenOut
	}

	return new(big.Int).Sub(amount, amountIn)
}

func cycleKey(path []triEdge) string {
	addrs := make([]string, len(path))
	for i, e := range path {
		addrs[i] = e.pool.Address().Hex()
	}
	sort.Strings(addrs)
	return strings.Join(addrs, "")
}
