package executor

import (
	"context"

	"github.com/dogukangundogan/trader/internal/strategy"
)

type Result struct {
	Success bool
	TxHash  string
	Profit  string
	Error   error
}

type Executor interface {
	Simulate(ctx context.Context, opp strategy.Opportunity) (bool, error)
	Execute(ctx context.Context, opp strategy.Opportunity) (*Result, error)
}
