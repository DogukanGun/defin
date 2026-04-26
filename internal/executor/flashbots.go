package executor

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/dogukangundogan/trader/internal/chain"
	"github.com/dogukangundogan/trader/internal/strategy"
)

const flashbotsRelayURL = "https://relay.flashbots.net"

type FlashbotsExecutor struct {
	client   *chain.Client
	privKey  *ecdsa.PrivateKey
	authKey  *ecdsa.PrivateKey
	from     common.Address
	chainID  *big.Int
	nonce    *chain.NonceTracker
	log      *slog.Logger
}

func NewFlashbotsExecutor(client *chain.Client, privKeyHex string, log *slog.Logger) (*FlashbotsExecutor, error) {
	privKey, err := crypto.HexToECDSA(privKeyHex)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	// Generate a separate auth key for Flashbots identity
	authKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("generate auth key: %w", err)
	}

	from := crypto.PubkeyToAddress(privKey.PublicKey)
	nonce := chain.NewNonceTracker(client, from)

	return &FlashbotsExecutor{
		client:  client,
		privKey: privKey,
		authKey: authKey,
		from:    from,
		chainID: big.NewInt(client.ChainID()),
		nonce:   nonce,
		log:     log,
	}, nil
}

func (f *FlashbotsExecutor) Simulate(ctx context.Context, opp strategy.Opportunity) (bool, error) {
	sim := NewSimulator(f.client, f.log)
	return sim.Simulate(ctx, opp)
}

func (f *FlashbotsExecutor) Execute(ctx context.Context, opp strategy.Opportunity) (*Result, error) {
	builder := NewBuilder()

	// Build transaction
	var calldata []byte
	var target common.Address
	var err error

	if len(opp.Steps) > 0 && opp.Steps[0].Pool != nil {
		calldata, err = builder.BuildSwapCalldata(opp.Steps[0])
		if err != nil {
			return &Result{Error: err}, err
		}
		target = opp.Steps[0].Pool.Address()
	}

	nonce, err := f.nonce.Next(ctx)
	if err != nil {
		return &Result{Error: err}, err
	}

	gasPrice, err := f.client.SuggestGasPrice(ctx)
	if err != nil {
		return &Result{Error: err}, err
	}

	tx := types.NewTransaction(nonce, target, big.NewInt(0), opp.GasEstimate, gasPrice, calldata)
	signer := types.NewEIP155Signer(f.chainID)
	signedTx, err := types.SignTx(tx, signer, f.privKey)
	if err != nil {
		return &Result{Error: err}, err
	}

	// Get current block for target block
	blockNum, err := f.client.BlockNumber(ctx)
	if err != nil {
		return &Result{Error: err}, err
	}

	txBytes, err := signedTx.MarshalBinary()
	if err != nil {
		return &Result{Error: err}, err
	}

	// Submit bundle
	err = f.submitBundle(ctx, []string{hexutil.Encode(txBytes)}, blockNum+1)
	if err != nil {
		f.nonce.Reset()
		return &Result{Error: err}, err
	}

	f.log.Info("flashbots bundle submitted",
		"tx_hash", signedTx.Hash().Hex(),
		"target_block", blockNum+1,
		"strategy", opp.StrategyName,
	)

	return &Result{
		Success: true,
		TxHash:  signedTx.Hash().Hex(),
		Profit:  opp.NetProfit.String(),
	}, nil
}

func (f *FlashbotsExecutor) submitBundle(ctx context.Context, txs []string, targetBlock uint64) error {
	params := map[string]interface{}{
		"txs":         txs,
		"blockNumber": fmt.Sprintf("0x%x", targetBlock),
	}

	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "eth_sendBundle",
		"params":  []interface{}{params},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", flashbotsRelayURL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	// Sign the payload with auth key for Flashbots identity
	signature, err := f.signPayload(body)
	if err != nil {
		return err
	}

	authAddr := crypto.PubkeyToAddress(f.authKey.PublicKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Flashbots-Signature", fmt.Sprintf("%s:%s", authAddr.Hex(), signature))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("flashbots request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("flashbots error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (f *FlashbotsExecutor) signPayload(body []byte) (string, error) {
	hash := crypto.Keccak256Hash(body)
	sig, err := crypto.Sign(hash.Bytes(), f.authKey)
	if err != nil {
		return "", err
	}
	return hexutil.Encode(sig), nil
}
