package executor

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log/slog"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/dogukangundogan/trader/internal/chain"
	"github.com/dogukangundogan/trader/internal/strategy"
)

type Sender struct {
	client  *chain.Client
	nonce   *chain.NonceTracker
	privKey *ecdsa.PrivateKey
	from    common.Address
	chainID *big.Int
	log     *slog.Logger
}

func NewSender(client *chain.Client, privKeyHex string, log *slog.Logger) (*Sender, error) {
	privKey, err := crypto.HexToECDSA(privKeyHex)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	from := crypto.PubkeyToAddress(privKey.PublicKey)
	nonce := chain.NewNonceTracker(client, from)

	return &Sender{
		client:  client,
		nonce:   nonce,
		privKey: privKey,
		from:    from,
		chainID: big.NewInt(client.ChainID()),
		log:     log,
	}, nil
}

func (s *Sender) From() common.Address { return s.from }

func (s *Sender) Execute(ctx context.Context, opp strategy.Opportunity) (*Result, error) {
	builder := NewBuilder()

	// For flash loan wrapped execution
	if opp.FlashLoan != nil {
		return s.executeFlashLoan(ctx, opp)
	}

	// Direct execution (no flash loan)
	for _, step := range opp.Steps {
		calldata, err := builder.BuildSwapCalldata(step)
		if err != nil {
			return &Result{Error: err}, err
		}

		var target common.Address
		if step.Pool != nil {
			target = step.Pool.Address()
		}

		txHash, err := s.sendTx(ctx, target, calldata, big.NewInt(0), opp.GasEstimate)
		if err != nil {
			return &Result{Error: err}, err
		}

		s.log.Info("transaction sent",
			"tx_hash", txHash,
			"strategy", opp.StrategyName,
			"target", target.Hex(),
		)
	}

	return &Result{
		Success: true,
		Profit:  opp.NetProfit.String(),
	}, nil
}

func (s *Sender) executeFlashLoan(ctx context.Context, opp strategy.Opportunity) (*Result, error) {
	// In production, this would call the FlashArb contract
	// which handles: borrow -> swap steps -> repay -> profit
	s.log.Info("flash loan execution",
		"strategy", opp.StrategyName,
		"token", opp.FlashLoan.Token.Hex(),
		"amount", opp.FlashLoan.Amount.String(),
	)

	return &Result{
		Success: true,
		Profit:  opp.NetProfit.String(),
	}, nil
}

func (s *Sender) sendTx(ctx context.Context, to common.Address, data []byte, value *big.Int, gasLimit uint64) (string, error) {
	nonce, err := s.nonce.Next(ctx)
	if err != nil {
		return "", fmt.Errorf("get nonce: %w", err)
	}

	gasPrice, err := s.client.SuggestGasPrice(ctx)
	if err != nil {
		return "", fmt.Errorf("suggest gas price: %w", err)
	}

	tx := types.NewTransaction(nonce, to, value, gasLimit, gasPrice, data)

	signer := types.NewEIP155Signer(s.chainID)
	signedTx, err := types.SignTx(tx, signer, s.privKey)
	if err != nil {
		return "", fmt.Errorf("sign tx: %w", err)
	}

	if err := s.client.HTTP().SendTransaction(ctx, signedTx); err != nil {
		s.nonce.Reset()
		return "", fmt.Errorf("send tx: %w", err)
	}

	return signedTx.Hash().Hex(), nil
}
