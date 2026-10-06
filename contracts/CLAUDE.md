# Payminto Smart Contracts — CLAUDE.md

## What This Is

The Payminto contracts are the on-chain component of the SmartSweep system — a set of Solidity contracts that enable deterministic deposit address generation and automated sweeping of funds to an immutable cold wallet. Three contracts work together: `AddressFactory` creates per-payment deposit addresses via CREATE2, `DepositProxy` is the minimal proxy that receives funds, and `SmartSweep` batches those funds into the merchant's cold storage. The project is built with Foundry (forge/cast/anvil).

## Tech Stack

| Component | Version |
|-----------|---------|
| Toolchain | Foundry (forge, cast, anvil) |
| Language | Solidity ^0.8.24 |
| Optimizer | 200 runs |
| Fuzz runs | 256 per test |
| OZ contracts | `lib/openzeppelin-contracts/` |
| Remapping | `@openzeppelin/=lib/openzeppelin-contracts/` |

## Directory Layout

```
contracts/
├── src/
│   ├── SmartSweep.sol        # Core: sweeps ETH/ERC20 to cold wallet, pause-able
│   ├── AddressFactory.sol    # CREATE2 factory: deploy DepositProxy per payment
│   └── DepositProxy.sol      # Minimal proxy: receives ETH, factory-controlled execute()
├── test/
│   ├── SmartSweep.t.sol      # 18 tests incl. fuzz; covers sweep, pause, access control
│   ├── AddressFactory.t.sol  # CREATE2 determinism, address prediction
│   └── Gas.t.sol             # Gas cost snapshots for deploy + sweep
├── script/
│   └── (Deploy.s.sol — not yet implemented)
├── lib/
│   └── openzeppelin-contracts/   # OZ v5, installed via forge install
├── out/                          # Compiled artifacts (forge build output)
├── cache/                        # Forge build cache
├── foundry.toml                  # Project config
└── remappings.txt                # @openzeppelin/ → lib/openzeppelin-contracts/
```

## Contract Architecture

### SmartSweep.sol

The central sweep contract. Key properties:
- `coldWallet` is set **immutably in the constructor** — it cannot be changed after deployment. Even if the sweeper key is compromised, an attacker cannot redirect funds to a different wallet.
- Only the `sweeper` role (a hot key controlled by the backend) can call `sweep()`.
- The owner (merchant) can pause/unpause the contract via `Pausable`.
- Supports both native ETH sweeps and ERC-20 token sweeps.
- Does not hold custody — it immediately forwards to `coldWallet`.

```solidity
constructor(address _coldWallet, address _sweeper) {
    coldWallet = _coldWallet; // immutable
    sweeper = _sweeper;
}

function sweep(address[] calldata tokens) external onlySweeper whenNotPaused {
    // send ETH balance to coldWallet
    // for each token: transfer full balance to coldWallet
}
```

### AddressFactory.sol

CREATE2 deterministic factory. Given a `salt` (derived from payment ID), it deploys a `DepositProxy` at a predictable address. The backend can compute the deposit address off-chain before the contract is deployed — the address exists as soon as the customer needs to pay, and the actual proxy is deployed only when sweeping.

```solidity
function getAddress(bytes32 salt) external view returns (address);
function deploy(bytes32 salt) external returns (address proxy);
function execute(address proxy, bytes calldata data) external onlyOwner;
```

### DepositProxy.sol

An ultra-minimal contract that:
- Receives ETH (via `receive()`)
- Allows the factory owner to call `execute()` to forward funds out
- Has no logic of its own — it is a pure fund receiver

## Common Commands

```bash
# Build all contracts
forge build

# Run all tests with verbosity
forge test -vvv

# Run a specific test file
forge test --match-path test/SmartSweep.t.sol -vvv

# Run a specific test function
forge test --match-test testSweepETH -vvv

# Run fuzz tests with more runs
forge test --fuzz-runs 1000

# Format Solidity source
forge fmt

# Check formatting without changing files
forge fmt --check

# Get gas report
forge test --gas-report

# Deploy (when Deploy.s.sol is implemented)
forge script script/Deploy.s.sol --rpc-url $RPC_URL --broadcast --verify

# Check a specific address on-chain
cast call $CONTRACT_ADDRESS "coldWallet()(address)" --rpc-url $RPC_URL

# Foundry binary location (if shadowed by Laravel Forge or other tools)
~/.foundry/bin/forge
~/.foundry/bin/cast
~/.foundry/bin/anvil
```

**Note:** If `forge` is not found, run `~/.foundry/bin/forge` directly. Laravel Forge (a PHP deployment tool) installs a `forge` binary at `/usr/local/bin/forge` which shadows Foundry's `forge`. Add `~/.foundry/bin` early in your PATH.

## Test Conventions

Tests use Foundry's standard test base:

```solidity
import {Test} from "forge-std/Test.sol";
import {SmartSweep} from "../src/SmartSweep.sol";

contract SmartSweepTest is Test {
    SmartSweep sweep;
    address coldWallet = makeAddr("cold");
    address sweeper = makeAddr("sweeper");

    function setUp() public {
        sweep = new SmartSweep(coldWallet, sweeper);
    }

    function testSweepETH() public {
        vm.deal(address(sweep), 1 ether);
        vm.prank(sweeper);
        sweep.sweep(new address[](0));
        assertEq(coldWallet.balance, 1 ether);
    }

    // Fuzz test example
    function testFuzz_sweepAmount(uint256 amount) public {
        vm.assume(amount > 0 && amount < 100_000 ether);
        // ...
    }
}
```

Use `vm.prank()` for single-call impersonation, `vm.startPrank()` for multi-call. Use `makeAddr()` for deterministic test addresses. Use `vm.deal()` for ETH and `deal(token, addr, amount)` for ERC-20 balances.

## Security Model

The security of SmartSweep rests on one key invariant: **the cold wallet address is immutable**. This means:

- If the backend (sweeper key) is compromised: the attacker can call `sweep()`, but funds still go to `coldWallet`. No theft possible from sweep.
- If the owner key is compromised: the attacker can pause the contract (griefing) but cannot redirect funds.
- To change the cold wallet, you must deploy a new `SmartSweep` contract and update the backend config.

Always verify that `coldWallet` in the deployed contract matches the expected address before authorising any sweeps.

## Integration Points

| System | Direction | Details |
|--------|-----------|---------|
| Go backend (`internal/blockchain/ethereum/`) | calls contracts | Calls `AddressFactory.getAddress()` to predict deposit address, calls `SmartSweep.sweep()` to batch-sweep |
| Merchant's cold wallet | receives funds | Immutably configured at deployment time |
| EVM chains | deployed on | Ethereum mainnet, Base, Polygon (same bytecode, different deployments) |

The backend service that interacts with these contracts is `internal/blockchain/ethereum/` in the Go backend. It uses `go-ethereum` (ethclient) to call contract methods.

## Key Files to Read First

1. `foundry.toml` — compiler settings, optimizer, fuzz config
2. `src/SmartSweep.sol` — the primary contract with security-critical immutability
3. `src/AddressFactory.sol` — CREATE2 factory, understand salt derivation
4. `test/SmartSweep.t.sol` — test coverage shows all expected behaviours
5. `remappings.txt` — import path configuration for OpenZeppelin
