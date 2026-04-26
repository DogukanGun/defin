package pool

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/dogukangundogan/trader/internal/config"
)

func FromConfig(cfg config.PoolConfig) (Pool, error) {
	addr := common.HexToAddress(cfg.Address)
	token0 := common.HexToAddress(cfg.Token0)
	token1 := common.HexToAddress(cfg.Token1)

	switch PoolType(cfg.Type) {
	case TypeUniswapV2:
		return NewUniswapV2Pool(addr, token0, token1, cfg.ChainID, cfg.FeeBps), nil
	case TypeUniswapV3:
		return NewUniswapV3Pool(addr, token0, token1, cfg.ChainID, cfg.FeeBps), nil
	case TypeCurve:
		return NewCurvePool(addr, cfg.ChainID, []common.Address{token0, token1}, cfg.FeeBps), nil
	default:
		return nil, fmt.Errorf("unknown pool type: %s", cfg.Type)
	}
}
