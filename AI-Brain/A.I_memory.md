- **Domain Separation**: Logic is now split across specialized services (Battle, Economy, Club, Employment, Oracle) to reduce `Lobby` mutex contention.

## Active Priorities
1. **Mainnet Secrets**: Secure wiring of `FAUCET_MNEMONIC` and `ADMIN_WALLETS` for production launch.
2. **Tournament Stress Testing**: Execute 16-player automated bracket tests to verify on-chain result archiving.

## Completed Tasks
- **Documentation**: Merged `ReadMe.txt` into `README.md`.
- **Hardening**: Applied `sync.RWMutex` to `main.go` (Plan F) and refactored async fetch logic.
- **Validation**: Performed simulated 16-player tournament stress tests; verified bracket archival and kickback logic.
- **Rewards**: Hardened Top 5 placement identification and implemented atomic multi-asset distribution.
- **Maintenance**: Verified `cleanupNonces` correctly prunes history without affecting active spectators.
- **Audit**: Verified EVM `power_divisor` and `power_base` configurations.
- **Visuals**: Implemented canvas-based particle effects for card captures (Phase 2).
- **SCSS Refactor**: Integrated `.particle-canvas` styles into the modular utility system.
- **Criminality**: Hardened heist logic to utilize `GetEffectiveCunning` (including faceplate bonuses) for success probability and kidnap eligibility.
- **Frontend Display**: Implemented display of Cunning and Nurturing values in `syncUI` with Cyberpunk styling.
- **Social UI**: Implemented the Social Hub (Alliances, Career, Achievements) using orphaned `_social.scss` styles.
- **Criminality UI**: Wired the Kidnap Selection interface following successful heists using orphaned `_criminality.scss` styles.
- **Shop UI Refactor**: Fully wired the District Shops overlay using orphaned `_shops.scss` styles, including category filtering.
- **Heist Planning Terminal**: Implemented full tactical interface for heists using `_criminality.scss` grid and risk meter styles.
- **Territory Map UI**: Implemented the 3D territory map visualization in `app.js` using the orphaned `_territory.scss` styles.
- **Heist Planning UI**: Implemented the Heist Planning interface in `app.js` using orphaned `_criminality.scss` styles.
- **Rewards**: Hardened `payoutAddress` validation in `faucet_service.go` to handle granular asset opt-ins.
- **Economy Audit**: Ensured loan interest and auction commissions are added to `l.faucetBalance` before dynamic scaling.
- **Stability**: Resolved critical deadlocks between `economy_processing.go` and `lobby_manager.go`.
- **Heist Logic Alignment**: Modified `handleHeist` in `club_service.go` to precisely match frontend heuristic by temporarily removing trap modifier calculation.
- **Consignment flow**: Implemented the auction creation interface in `app.js` using `_economy.scss` styles.
- **Economy UI**: Refactored Entity Market and Black Market UI; implemented Art Gallery (Auctions) using orphaned `_economy.scss` styles.
- **Social UI**: Implemented the Social Hub (Alliances, Career, Achievements) using orphaned `_social.scss` styles.
- **UI Immersion**: Implemented 3D Territory Map, Heist Planning Terminal, and Kidnap Selection overlays using legacy SCSS features.
- **Liquidity**: Implemented "Industrial Loop" recovery where black market scavenge fees return to the Faucet pool.
