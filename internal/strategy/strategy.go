package strategy

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/dogukangundogan/trader/internal/pool"
)

type Strategy interface {
	Name() string
	Evaluate(ctx context.Context, registry *pool.Registry, blockNumber uint64) ([]Opportunity, error)
	Enabled() bool
}

type Opportunity struct {
	StrategyName string
	ChainID      int64
	GrossProfit  *big.Int
	NetProfit    *big.Int
	GasEstimate  uint64
	Steps        []SwapStep
	FlashLoan    *FlashLoanParams
}

type SwapStep struct {
	Pool     pool.Pool
	TokenIn  common.Address
	TokenOut common.Address
	AmountIn *big.Int
}

type FlashLoanParams struct {
	Token    common.Address
	Amount   *big.Int
	Provider string // "aave_v3"
}
