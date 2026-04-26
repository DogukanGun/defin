package engine

import (
	"context"
	"log/slog"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/dogukangundogan/trader/internal/chain"
	"github.com/dogukangundogan/trader/internal/executor"
	"github.com/dogukangundogan/trader/internal/pool"
	"github.com/dogukangundogan/trader/internal/strategy"
	"github.com/dogukangundogan/trader/internal/telemetry"
)

// minUpdateInterval is the minimum time between multicall pool state refreshes.
// Arbitrum produces ~4 blocks/s; without throttling, we'd hit free RPC rate limits.
const minUpdateInterval = 1500 * time.Millisecond

// OpportunityNotifyFn is called whenever a profitable opportunity passes simulation.
// The caller may use it to broadcast structured events to connected clients.
type OpportunityNotifyFn func(opp strategy.Opportunity, chainName string)

type Engine struct {
	client      *chain.Client
	registry    *pool.Registry
	evaluator   *strategy.Evaluator
	executor    executor.Executor
	mode        string // "simulate" or "execute"
	log         *slog.Logger
	lastUpdate  atomic.Int64 // unix nanoseconds of last successful pool state update
	notifyOpp   OpportunityNotifyFn
}

func New(client *chain.Client, registry *pool.Registry, evaluator *strategy.Evaluator, exec executor.Executor, mode string, log *slog.Logger, notifyOpp OpportunityNotifyFn) *Engine {
	return &Engine{
		client:    client,
		registry:  registry,
		evaluator: evaluator,
		executor:  exec,
		mode:      mode,
		log:       log,
		notifyOpp: notifyOpp,
	}
}

func (e *Engine) HandleBlock(ctx context.Context, header *types.Header) error {
	start := time.Now()
	blockNum := header.Number.Uint64()
	chainName := e.client.Name()

	e.log.Debug("processing block", "chain", chainName, "block", blockNum)

	// Rate-limit pool state refreshes to avoid overwhelming the RPC endpoint.
	lastNs := e.lastUpdate.Load()
	if time.Since(time.Unix(0, lastNs)) >= minUpdateInterval {
		if err := e.updatePoolStates(ctx, header.Number); err != nil {
			e.log.Debug("failed to update pool states", "error", err, "block", blockNum)
			return err
		}
		e.lastUpdate.Store(time.Now().UnixNano())
		pools := e.registry.GetByChain(e.client.ChainID())
		e.log.Info("scanning pools for arbitrage", "chain", chainName, "block", blockNum, "pools", len(pools))
	}

	// Evaluate all strategies concurrently
	opps := e.evaluator.Evaluate(ctx, e.registry, blockNum)

	telemetry.BlocksProcessed.WithLabelValues(chainName).Inc()

	if len(opps) == 0 {
		elapsed := time.Since(start)
		telemetry.BlockProcessingDuration.WithLabelValues(chainName).Observe(elapsed.Seconds())
		return nil
	}

	e.log.Info("opportunities found",
		"chain", chainName,
		"block", blockNum,
		"count", len(opps),
		"best_net_profit", opps[0].NetProfit.String(),
	)

	for _, opp := range opps {
		telemetry.OpportunitiesFound.WithLabelValues(chainName, opp.StrategyName).Inc()
	}

	best := opps[0]
	if best.NetProfit.Sign() <= 0 {
		return nil
	}

	ok, err := e.executor.Simulate(ctx, best)
	if err != nil {
		e.log.Error("simulation error", "error", err, "strategy", best.StrategyName)
		return nil
	}
	if !ok {
		e.log.Info("simulation failed, skipping", "strategy", best.StrategyName)
		telemetry.OpportunitiesExecuted.WithLabelValues(chainName, best.StrategyName, "sim_failed").Inc()
		return nil
	}

	// Notify external listeners (e.g. WebSocket hub) with structured opportunity data.
	if e.notifyOpp != nil {
		e.notifyOpp(best, chainName)
	}

	if e.mode == "execute" {
		result, err := e.executor.Execute(ctx, best)
		if err != nil {
			e.log.Error("execution failed", "error", err, "strategy", best.StrategyName)
			telemetry.OpportunitiesExecuted.WithLabelValues(chainName, best.StrategyName, "failed").Inc()
			return nil
		}
		if result.Success {
			e.log.Info("opportunity executed",
				"strategy", best.StrategyName,
				"tx_hash", result.TxHash,
				"profit", result.Profit,
			)
			telemetry.OpportunitiesExecuted.WithLabelValues(chainName, best.StrategyName, "success").Inc()
			telemetry.ProfitTotal.WithLabelValues(chainName, best.StrategyName).Add(float64(best.NetProfit.Int64()))
		}
	} else {
		e.log.Info("SIMULATE: would execute",
			"strategy", best.StrategyName,
			"net_profit", best.NetProfit.String(),
			"gas", best.GasEstimate,
		)
		telemetry.OpportunitiesExecuted.WithLabelValues(chainName, best.StrategyName, "simulated").Inc()
	}

	elapsed := time.Since(start)
	telemetry.BlockProcessingDuration.WithLabelValues(chainName).Observe(elapsed.Seconds())
	e.log.Debug("block processed", "chain", chainName, "block", blockNum, "duration", elapsed)

	return nil
}

func (e *Engine) updatePoolStates(ctx context.Context, blockNumber *big.Int) error {
	pools := e.registry.GetByChain(e.client.ChainID())
	if len(pools) == 0 {
		return nil
	}

	calls := make([]chain.Call3, len(pools))
	for i, p := range pools {
		calls[i] = chain.Call3{
			Target:       p.Address(),
			AllowFailure: true,
			CallData:     p.StateCalldata(),
		}
	}

	mcStart := time.Now()
	results, err := e.client.Multicall(ctx, calls, blockNumber)
	telemetry.MulticallDuration.WithLabelValues(e.client.Name()).Observe(time.Since(mcStart).Seconds())

	if err != nil {
		return err
	}

	for i, result := range results {
		if !result.Success {
			continue
		}
		state, err := pools[i].DecodeState(result.ReturnData)
		if err != nil {
			e.log.Debug("decode state failed", "pool", pools[i].Address().Hex(), "error", err)
			continue
		}
		state.BlockNumber = blockNumber.Uint64()
		pools[i].UpdateState(state)
	}

	// Second pass: fetch liquidity() for V3 pools (slot0 doesn't include it).
	type v3LiqFetcher interface {
		LiquidityCalldata() []byte
		DecodeLiquidity([]byte) (*big.Int, error)
	}
	var v3Pools []pool.Pool
	var liqCalls []chain.Call3
	for _, p := range pools {
		if lf, ok := p.(v3LiqFetcher); ok {
			liqCalls = append(liqCalls, chain.Call3{
				Target:       p.Address(),
				AllowFailure: true,
				CallData:     lf.LiquidityCalldata(),
			})
			v3Pools = append(v3Pools, p)
		}
	}
	if len(liqCalls) > 0 {
		liqResults, err := e.client.Multicall(ctx, liqCalls, blockNumber)
		if err == nil {
			for i, r := range liqResults {
				if !r.Success {
					continue
				}
				liq, err := v3Pools[i].(v3LiqFetcher).DecodeLiquidity(r.ReturnData)
				if err != nil {
					continue
				}
				if st := v3Pools[i].State(); st != nil {
					st.Liquidity = liq
					v3Pools[i].UpdateState(st)
				}
			}
		}
	}

	return nil
}
