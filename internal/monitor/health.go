package monitor

import (
	"context"
	"log/slog"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/dogukangundogan/trader/internal/chain"
)

var (
	// getUserAccountData(address) selector
	getUserAccountDataSelector = crypto.Keccak256([]byte("getUserAccountData(address)"))[:4]
)

type Account struct {
	Address          common.Address
	ChainID          int64
	HealthFactor     *big.Int
	DebtToken        common.Address
	CollateralToken  common.Address
	DebtAmount       *big.Int
	CollateralAmount *big.Int
	LiquidationBonus int // in bps, e.g., 500 = 5%
}

type HealthMonitor struct {
	client       *chain.Client
	lendingPool  common.Address
	mu           sync.RWMutex
	accounts     map[common.Address]*Account
	watchList    []common.Address
	log          *slog.Logger
}

func NewHealthMonitor(client *chain.Client, lendingPool common.Address, log *slog.Logger) *HealthMonitor {
	return &HealthMonitor{
		client:      client,
		lendingPool: lendingPool,
		accounts:    make(map[common.Address]*Account),
		log:         log,
	}
}

func (m *HealthMonitor) AddWatch(addr common.Address) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.watchList = append(m.watchList, addr)
}

func (m *HealthMonitor) Update(ctx context.Context, blockNumber *big.Int) error {
	m.mu.RLock()
	addrs := make([]common.Address, len(m.watchList))
	copy(addrs, m.watchList)
	m.mu.RUnlock()

	if len(addrs) == 0 {
		return nil
	}

	// Build multicall to fetch all account data
	calls := make([]chain.Call3, len(addrs))
	for i, addr := range addrs {
		calldata := make([]byte, 36)
		copy(calldata[0:4], getUserAccountDataSelector)
		copy(calldata[16:36], addr.Bytes())

		calls[i] = chain.Call3{
			Target:       m.lendingPool,
			AllowFailure: true,
			CallData:     calldata,
		}
	}

	results, err := m.client.Multicall(ctx, calls, blockNumber)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for i, result := range results {
		if !result.Success || len(result.ReturnData) < 192 {
			continue
		}

		data := result.ReturnData
		// getUserAccountData returns:
		// (uint256 totalCollateralBase, uint256 totalDebtBase, uint256 availableBorrowsBase,
		//  uint256 currentLiquidationThreshold, uint256 ltv, uint256 healthFactor)
		healthFactor := new(big.Int).SetBytes(data[160:192])

		acct, exists := m.accounts[addrs[i]]
		if !exists {
			acct = &Account{
				Address:          addrs[i],
				ChainID:          m.client.ChainID(),
				LiquidationBonus: 500, // Default 5%, should be fetched per asset
			}
			m.accounts[addrs[i]] = acct
		}

		acct.HealthFactor = healthFactor
		acct.CollateralAmount = new(big.Int).SetBytes(data[0:32])
		acct.DebtAmount = new(big.Int).SetBytes(data[32:64])
	}

	return nil
}

func (m *HealthMonitor) GetLiquidatable() []Account {
	m.mu.RLock()
	defer m.mu.RUnlock()

	one := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil) // 1e18
	var result []Account
	for _, acct := range m.accounts {
		if acct.HealthFactor != nil && acct.HealthFactor.Cmp(one) < 0 {
			result = append(result, *acct)
		}
	}
	return result
}
