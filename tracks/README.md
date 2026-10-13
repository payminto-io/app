# Crypto World's Fair tracks

Payminto's current showcase focuses on Solana and Tempo for [Colosseum's Crypto World's Fair](https://colosseum.com/worldsfair).
See the [project details and prior-work disclosure](../docs/WORLDS_FAIR.md), [demo walkthrough](../DEMO.md), and [all product URLs](../website.md).

| Track | Implementation | Evidence boundary |
| --- | --- | --- |
| [Solana](solana/README.md) | USDC/USDT acceptance, shared ledger posting, and recoverable sweeps. | Public test USDC plus recorded fixtures and local-validator tests. |
| [Tempo](tempo/README.md) | TIP-20 invoice memos and wallet checkout on Moderato testnet. | Original owner-run real payment through the local backend; subsequent local QA; latest public release unverified. |

Earlier Chainlink, AWS, and NOWNodes submission write-ups have been removed from this track index.
Their underlying implementation and general architecture material remain in the repository.
Card/UPI connectors, custody, routing, and conversion remain unfinished or unmerged.
