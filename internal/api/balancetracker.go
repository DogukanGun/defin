package api

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/dogukangundogan/trader/internal/chain"
)

// knownTokens lists ERC-20 tokens to track on Arbitrum.
var knownTokens = []TokenBalance{
	{Symbol: "WETH",   Address: "0x82aF49447D8a07e3bd95BD0d56f35241523fBab1", Decimals: 18},
	{Symbol: "USDC.e", Address: "0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8", Decimals: 6},
	{Symbol: "USDT",   Address: "0xFd086bC7CD5C481DCC9C85ebE478A1C0b69FCbb9", Decimals: 6},
	{Symbol: "ARB",    Address: "0x912CE59144191C1204E64559FE8253a0e49E6548", Decimals: 18},
	{Symbol: "WBTC",   Address: "0x2f2a2543B76A4166549F7aaB2e75Bef0aefC5B0f", Decimals: 8},
	{Symbol: "USDC",   Address: "0xaf88d065e77c8cC2239327C5EDb3A432268e5831", Decimals: 6},
}

// balanceOfSelector is the ABI selector for balanceOf(address).
var balanceOfSelector = crypto.Keccak256([]byte("balanceOf(address)"))[:4]

func balanceOfCalldata(owner common.Address) []byte {
	data := make([]byte, 36)
	copy(data[:4], balanceOfSelector)
	copy(data[16:], owner.Bytes()) // ABI: address right-aligned in 32-byte slot
	return data
}

// StartBalanceTracker polls wallet token balances every interval and broadcasts
// them via the hub. It returns immediately; polling runs in a goroutine.
func StartBalanceTracker(ctx context.Context, privateKeyHex string, client *chain.Client, hub *Hub, interval time.Duration) {
	privKey, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
	if err != nil {
		// No valid private key (e.g. env not set) — skip silently
		return
	}
	walletAddr := crypto.PubkeyToAddress(privKey.PublicKey)

	go func() {
		poll := func() {
			eth := client.HTTP()
			if eth == nil {
				return
			}
			balances := fetchBalances(ctx, eth, walletAddr, client)
			hub.Send(Event{
				Type:          EventBalance,
				Chain:         client.Name(),
				WalletAddress: walletAddr.Hex(),
				Balances:      balances,
			})
		}

		poll() // immediate first fetch
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				poll()
			}
		}
	}()
}

func fetchBalances(ctx context.Context, eth *ethclient.Client, wallet common.Address, client *chain.Client) []TokenBalance {
	// Build multicall batch: one call per ERC-20
	calls := make([]chain.Call3, len(knownTokens))
	for i, t := range knownTokens {
		calls[i] = chain.Call3{
			Target:       common.HexToAddress(t.Address),
			AllowFailure: true,
			CallData:     balanceOfCalldata(wallet),
		}
	}

	results, err := client.Multicall(ctx, calls, nil)
	out := make([]TokenBalance, 0, len(knownTokens)+1)

	// Native ETH balance
	ethBal, ethErr := eth.BalanceAt(ctx, wallet, nil)
	if ethErr == nil {
		out = append(out, TokenBalance{
			Symbol:   "ETH",
			Address:  "native",
			RawWei:   ethBal.String(),
			Decimals: 18,
		})
	}

	if err != nil {
		return out
	}

	for i, r := range results {
		tb := TokenBalance{
			Symbol:   knownTokens[i].Symbol,
			Address:  knownTokens[i].Address,
			Decimals: knownTokens[i].Decimals,
		}
		if r.Success && len(r.ReturnData) >= 32 {
			tb.RawWei = new(big.Int).SetBytes(r.ReturnData[:32]).String()
		} else {
			tb.RawWei = "0"
		}
		out = append(out, tb)
	}
	return out
}
