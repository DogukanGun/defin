package strategy

import (
	"context"
	"log/slog"
	"sort"
	"sync"

	"github.com/dogukangundogan/trader/internal/pool"
)

type Evaluator struct {
	strategies []Strategy
	log        *slog.Logger
}

func NewEvaluator(strategies []Strategy, log *slog.Logger) *Evaluator {
	return &Evaluator{strategies: strategies, log: log}
}

func (e *Evaluator) Evaluate(ctx context.Context, registry *pool.Registry, blockNumber uint64) []Opportunity {
	var (
		mu   sync.Mutex
		opps []Opportunity
		wg   sync.WaitGroup
	)

	for _, s := range e.strategies {
		if !s.Enabled() {
			continue
		}

		wg.Add(1)
		go func(strat Strategy) {
			defer wg.Done()

			results, err := strat.Evaluate(ctx, registry, blockNumber)
			if err != nil {
				e.log.Error("strategy evaluation failed",
					"strategy", strat.Name(),
					"block", blockNumber,
					"error", err,
				)
				return
			}

			if len(results) > 0 {
				mu.Lock()
				opps = append(opps, results...)
				mu.Unlock()
			}
		}(s)
	}

	wg.Wait()

	// Sort by net profit descending
	sort.Slice(opps, func(i, j int) bool {
		return opps[i].NetProfit.Cmp(opps[j].NetProfit) > 0
	})

	return opps
}
