package chain

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Multicall3 aggregate3 signature: aggregate3((address,bool,bytes)[])
var aggregate3Selector = crypto.Keccak256([]byte("aggregate3((address,bool,bytes)[])"))[:4]

type Call3 struct {
	Target       common.Address
	AllowFailure bool
	CallData     []byte
}

type Call3Result struct {
	Success    bool
	ReturnData []byte
}

func (c *Client) Multicall(ctx context.Context, calls []Call3, blockNumber *big.Int) ([]Call3Result, error) {
	if len(calls) == 0 {
		return nil, nil
	}

	multicallAddr := common.HexToAddress(c.cfg.Multicall3)
	calldata := encodeAggregate3(calls)

	msg := ethereum.CallMsg{
		To:   &multicallAddr,
		Data: calldata,
	}

	result, err := c.HTTP().CallContract(ctx, msg, blockNumber)
	if err != nil {
		return nil, fmt.Errorf("multicall: %w", err)
	}

	return decodeAggregate3Results(result)
}

func encodeAggregate3(calls []Call3) []byte {
	// ABI encode: aggregate3((address target, bool allowFailure, bytes callData)[])
	// Layout:
	//   4 bytes selector
	//   32 bytes offset to array
	//   32 bytes array length
	//   for each element: 32 bytes offset
	//   for each element: encoded tuple

	numCalls := len(calls)

	// Pre-calculate encoded tuples
	var encodedTuples [][]byte
	for _, call := range calls {
		encodedTuples = append(encodedTuples, encodeTuple(call))
	}

	// Calculate total size
	// selector(4) + offset(32) + length(32) + offsets(32*n) + tuples
	totalTupleSize := 0
	for _, t := range encodedTuples {
		totalTupleSize += len(t)
	}
	totalSize := 4 + 32 + 32 + 32*numCalls + totalTupleSize

	buf := make([]byte, totalSize)
	copy(buf[0:4], aggregate3Selector)

	// Offset to the array (always 0x20 = 32)
	buf[35] = 0x20

	// Array length
	big.NewInt(int64(numCalls)).FillBytes(buf[36+32-1 : 36+32])
	putUint256(buf[36:68], int64(numCalls))

	// Offsets to each tuple (relative to start of array data)
	offsetBase := 32 * numCalls // offsets section size
	currentOffset := offsetBase
	for i := range encodedTuples {
		putUint256(buf[68+32*i:68+32*(i+1)], int64(currentOffset))
		currentOffset += len(encodedTuples[i])
	}

	// Write tuples
	pos := 68 + 32*numCalls
	for _, t := range encodedTuples {
		copy(buf[pos:pos+len(t)], t)
		pos += len(t)
	}

	return buf
}

func encodeTuple(call Call3) []byte {
	// (address, bool, bytes)
	// address: 32 bytes (left-padded)
	// bool: 32 bytes
	// offset to bytes: 32 bytes (always 0x60 = 96)
	// bytes length: 32 bytes
	// bytes data: padded to 32 bytes

	paddedLen := ((len(call.CallData) + 31) / 32) * 32
	size := 32 + 32 + 32 + 32 + paddedLen

	buf := make([]byte, size)

	// address
	copy(buf[12:32], call.Target.Bytes())

	// allowFailure
	if call.AllowFailure {
		buf[63] = 1
	}

	// offset to bytes data
	putUint256(buf[64:96], 96) // 0x60

	// bytes length
	putUint256(buf[96:128], int64(len(call.CallData)))

	// bytes data
	copy(buf[128:128+len(call.CallData)], call.CallData)

	return buf
}

func decodeAggregate3Results(data []byte) ([]Call3Result, error) {
	if len(data) < 64 {
		return nil, fmt.Errorf("multicall response too short: %d bytes", len(data))
	}

	// Skip offset (32 bytes), read array length
	arrayLen := new(big.Int).SetBytes(data[32:64]).Int64()
	if arrayLen == 0 {
		return nil, nil
	}

	results := make([]Call3Result, arrayLen)

	// Read offsets to each result tuple
	offsets := make([]int64, arrayLen)
	for i := int64(0); i < arrayLen; i++ {
		start := 64 + 32*i
		offsets[i] = new(big.Int).SetBytes(data[start : start+32]).Int64()
	}

	// Decode each result: (bool success, bytes returnData)
	arrayDataStart := int64(64)
	for i := int64(0); i < arrayLen; i++ {
		tupleStart := arrayDataStart + offsets[i]
		if tupleStart+64 > int64(len(data)) {
			return nil, fmt.Errorf("result %d: out of bounds", i)
		}

		results[i].Success = data[tupleStart+31] == 1

		// Offset to bytes
		bytesOffset := new(big.Int).SetBytes(data[tupleStart+32 : tupleStart+64]).Int64()
		bytesLenPos := tupleStart + bytesOffset
		if bytesLenPos+32 > int64(len(data)) {
			return nil, fmt.Errorf("result %d: bytes length out of bounds", i)
		}

		bytesLen := new(big.Int).SetBytes(data[bytesLenPos : bytesLenPos+32]).Int64()
		bytesStart := bytesLenPos + 32
		if bytesStart+bytesLen > int64(len(data)) {
			return nil, fmt.Errorf("result %d: bytes data out of bounds", i)
		}

		results[i].ReturnData = make([]byte, bytesLen)
		copy(results[i].ReturnData, data[bytesStart:bytesStart+bytesLen])
	}

	return results, nil
}

func putUint256(buf []byte, val int64) {
	b := big.NewInt(val)
	bytes := b.Bytes()
	copy(buf[32-len(bytes):32], bytes)
}
