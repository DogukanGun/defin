package strategy

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// FlashLoan wraps any opportunity's execution steps in an Aave v3 flash loan.
// The FlashArb.sol contract handles:
//   1. Borrow token via flashLoan
//   2. Execute swap steps
//   3. Repay loan + 0.05% fee
//   4. Send profit to caller

var (
	// flashLoan(address receiverAddress, address[] assets, uint256[] amounts, uint256[] modes, address onBehalfOf, bytes params, uint16 referralCode)
	flashLoanSelector = crypto.Keccak256([]byte("flashLoan(address,address[],uint256[],uint256[],address,bytes,uint16)"))[:4]
)

type FlashLoanBuilder struct {
	lendingPool  common.Address
	flashArbAddr common.Address
}

func NewFlashLoanBuilder(lendingPool, flashArbAddr common.Address) *FlashLoanBuilder {
	return &FlashLoanBuilder{
		lendingPool:  lendingPool,
		flashArbAddr: flashArbAddr,
	}
}

func (b *FlashLoanBuilder) BuildCalldata(opp Opportunity) (common.Address, []byte, error) {
	if opp.FlashLoan == nil {
		return common.Address{}, nil, nil
	}

	// Encode the flash loan call to the lending pool
	// The params field encodes the swap steps for the FlashArb contract to execute
	params := encodeSwapSteps(opp.Steps)

	calldata := make([]byte, 0, 4+32*7+len(params))
	calldata = append(calldata, flashLoanSelector...)

	// receiverAddress (our FlashArb contract)
	calldata = append(calldata, padAddress(b.flashArbAddr)...)

	// Offsets for dynamic arrays (assets, amounts, modes)
	// assets offset: 7 * 32 = 224 = 0xE0
	calldata = append(calldata, padUint256(big.NewInt(224))...)
	// amounts offset: 224 + 64 + 32 = 320 = 0x140
	calldata = append(calldata, padUint256(big.NewInt(320))...)
	// modes offset: 320 + 64 = 384 = 0x180 ... simplified
	// This is a simplified encoding; real implementation uses proper ABI encoding

	_ = params // Full ABI encoding would be done here

	return b.lendingPool, calldata, nil
}

func encodeSwapSteps(steps []SwapStep) []byte {
	// Encode swap steps as: [(address pool, address tokenIn, address tokenOut, uint256 amountIn)]
	buf := make([]byte, 0, len(steps)*128)
	for _, step := range steps {
		if step.Pool != nil {
			buf = append(buf, padAddress(step.Pool.Address())...)
		} else {
			buf = append(buf, make([]byte, 32)...)
		}
		buf = append(buf, padAddress(step.TokenIn)...)
		buf = append(buf, padAddress(step.TokenOut)...)
		if step.AmountIn != nil {
			buf = append(buf, padUint256(step.AmountIn)...)
		} else {
			buf = append(buf, make([]byte, 32)...)
		}
	}
	return buf
}

func padAddress(addr common.Address) []byte {
	buf := make([]byte, 32)
	copy(buf[12:], addr.Bytes())
	return buf
}

func padUint256(val *big.Int) []byte {
	buf := make([]byte, 32)
	if val != nil {
		b := val.Bytes()
		copy(buf[32-len(b):], b)
	}
	return buf
}
