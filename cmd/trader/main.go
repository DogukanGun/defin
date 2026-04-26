package main

import (
	"context"
	"flag"
	"log/slog"
	"math/big"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/dogukangundogan/trader/internal/api"
	"github.com/dogukangundogan/trader/internal/chain"
	"github.com/dogukangundogan/trader/internal/config"
	"github.com/dogukangundogan/trader/internal/engine"
	"github.com/dogukangundogan/trader/internal/executor"
	"github.com/dogukangundogan/trader/internal/pool"
	"github.com/dogukangundogan/trader/internal/strategy"
	"github.com/dogukangundogan/trader/internal/telemetry"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Set up WebSocket hub and wrap logger so all log output streams to connected clients
	hub := api.NewHub()
	baseLog := telemetry.NewLogger(cfg.Telemetry.LogLevel)
	log := slog.New(api.NewHubHandler(baseLog.Handler(), hub))

	// Start metrics server
	go func() {
		if err := telemetry.ServeMetrics(cfg.Telemetry.MetricsPort); err != nil {
			log.Error("metrics server failed", "error", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Info("received shutdown signal", "signal", sig)
		cancel()
	}()

	// Initialize registry
	registry := pool.NewRegistry()
	for _, pc := range cfg.Pools {
		p, err := pool.FromConfig(pc)
		if err != nil {
			log.Error("failed to create pool", "address", pc.Address, "error", err)
			continue
		}
		registry.Add(p)
	}
	log.Info("pool registry initialized", "count", registry.Len())

	// Start API server (HTTP + WebSocket) — after registry is populated
	go func() {
		srv := api.NewServer(hub, registry, cfg, log)
		if err := srv.Start(":8080"); err != nil {
			log.Error("API server failed", "error", err)
		}
	}()

	// Connect to each chain and start engines
	for _, chainCfg := range cfg.Chains {
		chainCfg := chainCfg

		client, err := chain.NewClient(ctx, chain.ClientConfig{
			Name:       chainCfg.Name,
			ChainID:    chainCfg.ChainID,
			RPCHTTP:    chainCfg.RPCHTTP,
			RPCWS:      chainCfg.RPCWS,
			Multicall3: chainCfg.Multicall3,
		}, log)
		if err != nil {
			log.Error("failed to connect to chain", "chain", chainCfg.Name, "error", err)
			continue
		}
		defer client.Close()

		gasPrice := big.NewInt(int64(chainCfg.MaxGasPriceGwei * 1e9))
		minProfitWei := toWei(cfg.Strategies.CrossDex.MinProfitUSD)

		// Build strategies
		strategies := buildStrategies(cfg, gasPrice, minProfitWei, log)

		evaluator := strategy.NewEvaluator(strategies, log)

		// Build executor
		var exec executor.Executor
		if cfg.Execution.UseFlashbots && cfg.Execution.Mode == "execute" {
			privKey := strings.TrimPrefix(cfg.PrivateKey, "0x")
			exec, err = executor.NewFlashbotsExecutor(client, privKey, log)
			if err != nil {
				log.Error("failed to create flashbots executor", "error", err)
				continue
			}
		} else {
			exec = executor.NewSimulator(client, log)
		}

		notifyFn := func(opp strategy.Opportunity, chainName string) {
			steps := make([]api.EventStep, len(opp.Steps))
			for i, s := range opp.Steps {
				steps[i] = api.EventStep{
					TokenIn:  s.TokenIn.Hex(),
					TokenOut: s.TokenOut.Hex(),
					Pool:     s.Pool.Address().Hex(),
					PoolType: string(s.Pool.Type()),
				}
			}
			hub.Send(api.Event{
				Type:        api.EventOpportunity,
				Strategy:    opp.StrategyName,
				Chain:       chainName,
				NetProfit:   opp.NetProfit.String(),
				GasEstimate: opp.GasEstimate,
				Steps:       steps,
			})
		}

		// Start balance tracker for this chain (Arbitrum only, once)
		if chainCfg.ChainID == 42161 && cfg.PrivateKey != "" {
			api.StartBalanceTracker(ctx, cfg.PrivateKey, client, hub, 15*time.Second)
		}

		eng := engine.New(client, registry, evaluator, exec, cfg.Execution.Mode, log, notifyFn)

		blockTime := time.Duration(chainCfg.BlockTimeMS) * time.Millisecond
		sub := chain.NewSubscriber(client, eng.HandleBlock, blockTime, log)
		if err := sub.Start(ctx); err != nil {
			log.Error("failed to start subscriber", "chain", chainCfg.Name, "error", err)
			continue
		}

		log.Info("engine started", "chain", chainCfg.Name, "mode", cfg.Execution.Mode)
	}

	// Block until shutdown
	<-ctx.Done()
	log.Info("trader shutdown complete")
}

func buildStrategies(cfg *config.Config, gasPrice, minProfitWei *big.Int, log *slog.Logger) []strategy.Strategy {
	var strategies []strategy.Strategy

	strategies = append(strategies, strategy.NewCrossDex(
		cfg.Strategies.CrossDex.Enabled,
		minProfitWei,
		gasPrice,
		log,
	))

	strategies = append(strategies, strategy.NewTriangular(
		cfg.Strategies.Triangular.Enabled,
		minProfitWei,
		gasPrice,
		cfg.Strategies.Triangular.MaxHops,
		log,
	))

	strategies = append(strategies, strategy.NewCurveStable(
		cfg.Strategies.CurveStable.Enabled,
		minProfitWei,
		gasPrice,
		log,
	))

	// Liquidation strategy requires health monitor setup
	// which needs Aave lending pool address per chain
	strategies = append(strategies, strategy.NewLiquidation(
		cfg.Strategies.Liquidation.Enabled,
		minProfitWei,
		gasPrice,
		nil, // health monitor initialized per-chain if enabled
		log,
	))

	return strategies
}

func toWei(usd float64) *big.Int {
	// Approximate: 1 USD ≈ 0.0003 ETH at ~$3000/ETH
	// minProfit in wei = usd * 0.0003 * 1e18
	ethPerUSD := 0.0003
	weiPerETH := 1e18
	wei := usd * ethPerUSD * weiPerETH
	return big.NewInt(int64(wei))
}
