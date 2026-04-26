// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

interface IPool {
    function flashLoan(
        address receiverAddress,
        address[] calldata assets,
        uint256[] calldata amounts,
        uint256[] calldata interestRateModes,
        address onBehalfOf,
        bytes calldata params,
        uint16 referralCode
    ) external;
}

interface IFlashLoanReceiver {
    function executeOperation(
        address[] calldata assets,
        uint256[] calldata amounts,
        uint256[] calldata premiums,
        address initiator,
        bytes calldata params
    ) external returns (bool);
}

interface IERC20 {
    function transfer(address to, uint256 amount) external returns (bool);
    function approve(address spender, uint256 amount) external returns (bool);
    function balanceOf(address account) external view returns (uint256);
}

interface IUniswapV2Pair {
    function swap(uint amount0Out, uint amount1Out, address to, bytes calldata data) external;
    function token0() external view returns (address);
}

/// @title FlashArb - Atomic flash loan arbitrage executor
/// @notice Borrows via Aave v3 flash loan, executes swap steps, repays, and sends profit to owner
contract FlashArb is IFlashLoanReceiver {
    address public immutable owner;
    address public immutable lendingPool;

    struct SwapStep {
        address pool;
        address tokenIn;
        address tokenOut;
        uint256 amountIn; // 0 means use full balance
    }

    constructor(address _lendingPool) {
        owner = msg.sender;
        lendingPool = _lendingPool;
    }

    modifier onlyOwner() {
        require(msg.sender == owner, "not owner");
        _;
    }

    /// @notice Initiates a flash loan arbitrage
    /// @param asset The token to borrow
    /// @param amount The amount to borrow
    /// @param params ABI-encoded SwapStep[] for execution
    function execute(address asset, uint256 amount, bytes calldata params) external onlyOwner {
        address[] memory assets = new address[](1);
        assets[0] = asset;

        uint256[] memory amounts = new uint256[](1);
        amounts[0] = amount;

        uint256[] memory modes = new uint256[](1);
        modes[0] = 0; // no debt, must repay in same tx

        IPool(lendingPool).flashLoan(
            address(this),
            assets,
            amounts,
            modes,
            address(this),
            params,
            0
        );
    }

    /// @notice Aave flash loan callback
    function executeOperation(
        address[] calldata assets,
        uint256[] calldata amounts,
        uint256[] calldata premiums,
        address initiator,
        bytes calldata params
    ) external override returns (bool) {
        require(msg.sender == lendingPool, "not lending pool");
        require(initiator == address(this), "not initiator");

        // Decode and execute swap steps
        SwapStep[] memory steps = abi.decode(params, (SwapStep[]));
        for (uint256 i = 0; i < steps.length; i++) {
            _executeStep(steps[i]);
        }

        // Repay flash loan (amount + premium)
        uint256 amountOwed = amounts[0] + premiums[0];
        IERC20(assets[0]).approve(lendingPool, amountOwed);

        // Send remaining profit to owner
        uint256 balance = IERC20(assets[0]).balanceOf(address(this));
        if (balance > amountOwed) {
            IERC20(assets[0]).transfer(owner, balance - amountOwed);
        }

        return true;
    }

    function _executeStep(SwapStep memory step) internal {
        uint256 amountIn = step.amountIn;
        if (amountIn == 0) {
            amountIn = IERC20(step.tokenIn).balanceOf(address(this));
        }

        // Approve and swap
        IERC20(step.tokenIn).approve(step.pool, amountIn);

        // Determine if this is a V2 pair swap
        // For V2: call swap(amount0Out, amount1Out, to, data)
        address token0 = IUniswapV2Pair(step.pool).token0();
        (uint256 amount0Out, uint256 amount1Out) = step.tokenOut == token0
            ? (amountIn, uint256(0))  // This is simplified; real impl would calculate output
            : (uint256(0), amountIn);

        // Transfer tokenIn to the pair first (V2 pattern)
        IERC20(step.tokenIn).transfer(step.pool, amountIn);
        IUniswapV2Pair(step.pool).swap(amount0Out, amount1Out, address(this), "");
    }

    /// @notice Emergency withdraw by owner
    function withdraw(address token) external onlyOwner {
        uint256 balance = IERC20(token).balanceOf(address(this));
        if (balance > 0) {
            IERC20(token).transfer(owner, balance);
        }
    }

    receive() external payable {}
}
