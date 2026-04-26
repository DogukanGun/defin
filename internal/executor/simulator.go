package executor

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"

	"github.com/dogukangundogan/trader/internal/chain"
	"github.com/dogukangundogan/trader/internal/strategy"
)

type Simulator struct {
	client *chain.Client
	log    *slog.Logger
}

func NewSimulator(client *chain.Client, log *slog.Logger) *Simulator {
	return &Simulator{client: client, log: log}
}

func (s *Simulator) Simulate(ctx context.Context, opp strategy.Opportunity) (bool, error) {
	if len(opp.Steps) == 0 {
		return false, fmt.Errorf("no swap steps")
	}

	builder := NewBuilder()

	for _, step := range opp.Steps {
		calldata, err := builder.BuildSwapCalldata(step)
		if err != nil {
			return false, fmt.Errorf("build calldata: %w", err)
		}

		var target common.Address
		if step.Pool != nil {
			target = step.Pool.Address()
		}

		msg := ethereum.CallMsg{
			To:   &target,
			Data: calldata,
		}

		_, err = s.client.HTTP().CallContract(ctx, msg, nil)
		if err != nil {
			s.log.Debug("simulation failed",
				"strategy", opp.StrategyName,
				"step_pool", target.Hex(),
				"error", err,
			)
			return false, nil
		}
	}

	s.log.Info("simulation passed",
		"strategy", opp.StrategyName,
		"net_profit", opp.NetProfit.String(),
	)
	return true, nil
}

func (s *Simulator) Execute(ctx context.Context, opp strategy.Opportunity) (*Result, error) {
	ok, err := s.Simulate(ctx, opp)
	if err != nil {
		return &Result{Error: err}, err
	}
	return &Result{
		Success: ok,
		Profit:  opp.NetProfit.String(),
	}, nil
}

func (s *Simulator) SimulateFlashLoan(ctx context.Context, opp strategy.Opportunity, flashArbAddr, lendingPool common.Address) (bool, error) {
	fb := strategy.NewFlashLoanBuilder(lendingPool, flashArbAddr)
	target, calldata, err := fb.BuildCalldata(opp)
	if err != nil {
		return false, err
	}

	value := big.NewInt(0)
	msg := ethereum.CallMsg{
		To:    &target,
		Data:  calldata,
		Value: value,
	}

	_, err = s.client.HTTP().CallContract(ctx, msg, nil)
	if err != nil {
		s.log.Debug("flash loan simulation failed",
			"strategy", opp.StrategyName,
			"error", err,
		)
		return false, nil
	}

	return true, nil
}
