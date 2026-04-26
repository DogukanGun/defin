package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Chains     []ChainConfig    `yaml:"chains"`
	Pools      []PoolConfig     `yaml:"pools"`
	Strategies StrategyConfig   `yaml:"strategies"`
	Execution  ExecutionConfig  `yaml:"execution"`
	PrivateKey string           `yaml:"private_key"`
	Telemetry  TelemetryConfig  `yaml:"telemetry"`
}

type ChainConfig struct {
	Name           string  `yaml:"name"`
	ChainID        int64   `yaml:"chain_id"`
	RPCHTTP        string  `yaml:"rpc_http"`
	RPCWS          string  `yaml:"rpc_ws"`
	Multicall3     string  `yaml:"multicall3"`
	BlockTimeMS    int     `yaml:"block_time_ms"`
	MaxGasPriceGwei float64 `yaml:"max_gas_price_gwei"`
}

type PoolConfig struct {
	Address string `yaml:"address"`
	Type    string `yaml:"type"`
	ChainID int64  `yaml:"chain_id"`
	Token0  string `yaml:"token0"`
	Token1  string `yaml:"token1"`
	FeeBps  int    `yaml:"fee_bps"`
}

type StrategyConfig struct {
	CrossDex    StrategyEntry `yaml:"cross_dex"`
	Triangular  StrategyEntry `yaml:"triangular"`
	CurveStable StrategyEntry `yaml:"curve_stable"`
	Liquidation StrategyEntry `yaml:"liquidation"`
}

type StrategyEntry struct {
	Enabled      bool    `yaml:"enabled"`
	MinProfitUSD float64 `yaml:"min_profit_usd"`
	MaxHops      int     `yaml:"max_hops,omitempty"`
}

type ExecutionConfig struct {
	Mode              string  `yaml:"mode"`
	UseFlashbots      bool    `yaml:"use_flashbots"`
	UseFlashLoan      bool    `yaml:"use_flash_loan"`
	FlashLoanProvider string  `yaml:"flash_loan_provider"`
	MaxPositionUSD    float64 `yaml:"max_position_usd"`
	SlippageBps       int     `yaml:"slippage_bps"`
}

type TelemetryConfig struct {
	LogLevel    string `yaml:"log_level"`
	MetricsPort int    `yaml:"metrics_port"`
}

var envVarRe = regexp.MustCompile(`\$\{([^}]+)\}`)

func expandEnvVars(s string) string {
	return envVarRe.ReplaceAllStringFunc(s, func(match string) string {
		key := strings.TrimSuffix(strings.TrimPrefix(match, "${"), "}")
		if val, ok := os.LookupEnv(key); ok {
			return val
		}
		return match
	})
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	expanded := expandEnvVars(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) validate() error {
	if len(c.Chains) == 0 {
		return fmt.Errorf("at least one chain must be configured")
	}
	for i, ch := range c.Chains {
		if ch.ChainID == 0 {
			return fmt.Errorf("chain[%d]: chain_id is required", i)
		}
		if ch.RPCHTTP == "" && ch.RPCWS == "" {
			return fmt.Errorf("chain[%d] %s: at least one RPC endpoint is required", i, ch.Name)
		}
	}
	if c.Execution.Mode != "simulate" && c.Execution.Mode != "execute" {
		return fmt.Errorf("execution.mode must be 'simulate' or 'execute', got %q", c.Execution.Mode)
	}
	return nil
}

func (c *Config) ChainByID(chainID int64) *ChainConfig {
	for i := range c.Chains {
		if c.Chains[i].ChainID == chainID {
			return &c.Chains[i]
		}
	}
	return nil
}
