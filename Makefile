.PHONY: build test run clean abigen lint

BINARY=trader
GO=go

build:
	$(GO) build -o bin/$(BINARY) ./cmd/trader

run: build
	./bin/$(BINARY) -config config.yaml

test:
	$(GO) test ./... -v -race -count=1

test-short:
	$(GO) test ./... -short -race -count=1

bench:
	$(GO) test ./internal/math/... -bench=. -benchmem

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/

abigen:
	@echo "Generating Go bindings from ABIs..."
	@mkdir -p pkg/contracts/uniswapv2pair
	@mkdir -p pkg/contracts/uniswapv3pool
	@mkdir -p pkg/contracts/curvepool
	@mkdir -p pkg/contracts/aavelendingpool
	@mkdir -p pkg/contracts/multicall3
	@mkdir -p pkg/contracts/erc20
	@mkdir -p pkg/contracts/uniswapv2router
	@mkdir -p pkg/contracts/uniswapv3quoter
	abigen --abi abi/UniswapV2Pair.json --pkg uniswapv2pair --out pkg/contracts/uniswapv2pair/uniswapv2pair.go
	abigen --abi abi/UniswapV3Pool.json --pkg uniswapv3pool --out pkg/contracts/uniswapv3pool/uniswapv3pool.go
	abigen --abi abi/CurveStableSwap.json --pkg curvepool --out pkg/contracts/curvepool/curvepool.go
	abigen --abi abi/AaveLendingPoolV3.json --pkg aavelendingpool --out pkg/contracts/aavelendingpool/aavelendingpool.go
	abigen --abi abi/Multicall3.json --pkg multicall3 --out pkg/contracts/multicall3/multicall3.go
	abigen --abi abi/ERC20.json --pkg erc20 --out pkg/contracts/erc20/erc20.go

tidy:
	$(GO) mod tidy
