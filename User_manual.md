## ⚔️ VIRTUALBABES ARENA## PLAYER'S USER MANUAL
------------------------------
## 1. GETTING STARTED## A. Connecting Your Wallet
Access the Arena by authenticating your Voi wallet.

* Nautilus Wallet
* Kibisis Wallet
* WalletConnect

Your public address acts as your unique player identity.
## B. Choosing Your Avatar

* Identity: Select any owned NFT to represent you in lobbies and matches.
* Gloat Message: Configure a custom taunt displayed automatically upon victory.

## C. Building Your Deck
Decks require exactly 5 cards. Manage assets via the Deck Manager:

* **Inventory Auto-Discovery**: The Arena automatically scans your wallet for compatible NFTs. We natively support **ARC-72** (Smart Contract), **ARC-19** (Dynamic/Reserve-based), and **ARC-69** (Note-based) standards. **Zero Configuration**: No manual import is required; your diverse collections appear instantly upon connection.
* **Legacy Support**: Older Algorand Standard Assets (ASAs) using transaction notes for metadata are fully ingested by our multi-standard Oracle.
* **Verification**: Assets are cryptographically verified against the blockchain indexer to ensure you are playing with authentic collectibles.
* Deck Slots: Unlock additional slots by increasing your Reputation.
* Auto-Build: Deploy system optimization to generate maximum power decks instantly.

------------------------------
## 2. COMBAT & MATCHMAKING## A. Matchmaking Queue
Enter the pool to pair against opponents. The algorithm matches players using two metrics:

* Reputation
* Deck Rating

## B. Combat Rules (Tactical 3x3 Grid)
Matches use a 3x3 grid. Each card features 4 directional power values: Top, Right, Bottom, Left.

      [Top]
[Left]     [Right]
    [Bottom]


* Basic Capture: Outscore an adjacent card's facing value to flip it.
* Same Rule: Match adjacent values on 2+ sides to capture all targets.
* Plus Rule: Equal the sum of adjacent values on 2+ sides to capture targets.
* Combo Chain: Flipped cards trigger cascading captures if their values beat neighbors.
* Elemental Synthesis: Match card Moods (Volatile, Serene, Spirited, Grounded) to tile Moods for buffs.
* Coalition Defense: Allied members defending a partner organization's territory receive a prioritized +10% power boost.
* Fallen Penalty: Captured cards lose 20 Artifact power immediately. This degradation is permanent and persists across the Arena ecosystem.
* Sudden Death: Ties (5-5) trigger a board clear. Remaining cards redistribute for a final tie-breaker.

## C. Player Attributes

* Mojo: Social rank. Unlocks premium items and buffs employee progression.
* Reputation: Arena standing. Directly dictates your asset price in the Entity Market.
* Wanted Level: Rises via crime. High levels apply power penalties during combat.
* Cunning: Boosts heist success rates. Mitigates active criminal penalties.
* Nurturing: Decreases the rate of card Fatigue accumulation.

------------------------------
## 3. THE INDUSTRIAL LOOP## A. Clubs & Territories

* Founding: Pay a fixed fee to establish a Club and claim open territory.
* Joining: Pay an entry fee to enlist in an established organization.
* Regions: Controlling 2+ territories upgrades your Club to a Region.
* Governors: Regional leaders grant members a +5% power buff in home districts.
* Mojo Accumulation: Earned via store revenue and successful heist defenses.

## B. Employment & Careers
Club owners delegate operational roles to hired players:

* Manager: Adjusts shop commission rates and controls inventory logs.
* Security: Deploys active sector traps and lowers enemy heist success.
* Clerk: Accelerates store turnover speeds to optimize revenue.

Employment builds Reputation. Contracts above 500 $VBV trigger the EXECUTIVE_PAY achievement.
* **Insolvency Protection**: If an employer club dissolves or defaults on payments, employee contracts are terminated, and players revert to **Freelancer** status. The salary clock resets to prevent timing exploits during re-hiring.
## C. Shops & Items
Clubs run localized storefronts supplying tactical utility items:

* Elemental Forge: Mood Catalysts, Grounded Shields, Prism Shields.
* Tactical Syndicate: Rule Breakers, Intel Reports, Ghost Protocol.
* Vitality Lab: Stamina Stims, Loyalty Pledges, Hyper-Stims.
* Hardware/Security: Laser Tripwires, Sentry Turrets, Bio-Guard Dogs.

Premium gear requires Master Tier (Governor status) or Role-Gating (Specific Job).
## D. Industrial Leases

* Lease Board: Rent out idle cards to other players for passive income.
* Revenue Split: 50% Lender | 20% Faucet Tax | 20% Club Treasury | 10% Club Members.
* Terminology: Leased cards automatically return to the owner's inventory upon expiration. Revenue splits utilize micro-unit precision math; remainders are rerouted to the Club Treasury to maintain the Industrial Seal.

------------------------------
## 4. HIGH-FINANCE & MARKETS

 💰 ENTITY MARKET  📈 ART GALLERY   🏦 SECOND-HAND   🥷 BLACK MARKET
  (Trade Shares)    (Asset Bundles)    (Card Loans)     (Discount Assets)


* Entity Market: Buy and sell shares of players and NPCs. Valuation shifts with wins and rumors.
* Art Gallery: Auction Card Bundles (Card + Weapon + Faceplate). Venues claim a 10% commission.
* Second-Hand Store: Collateralize cards for instant $VBV liquidity. Defaults prompt liquidation.
* Black Market: Restricted access (Wanted 5+ / Cunning 10+). Purchase defaulted cards at a discount.
* Rumor Mill: Pay to manipulate market sentiments. Spreading a rumor costs 500 $VBV and incurs a **20% Regional Governor Tax** distributed to district leaders.

------------------------------
## 5. CRIMINALITY & JUSTICE## A. Heists & Jailing
Target enemy Club Treasuries to steal capital. Success scales on Cunning vs. Security.

* Fence Fee: A 10% tax on successful hauls routes directly to the Faucet.
* Guard Dogs: Failed heists against protected clubs jail your rarest card.
* Bail: Pay 200 $VBV to the capturing Club to free jailed assets instantly.
* Penalties: Active jail sentences continuously drain your profile Reputation.

## B. Kidnap Gambit

* Hostage Protocol: High-tier heists allow players to hold a rare card for ransom.
* Target Selection: Perpetrators automatically target the victim's **Favorite Card** or their **Rarest Asset** to maximize leverage.
* Strategic Balance: A victim can only have **one** active hostage situation at a time. New kidnapping attempts will fail until the current situation is resolved.
* Ransom Tax: Victim payments face a 20% Laundering Tax sent to the Faucet.
* Insurance: Unpaid ransoms trigger auto-return safety protocols after 48 hours.
* Standing: Both victims and perpetrators suffer immediate Reputation shifts when a kidnapping occurs, reflecting the social volatility of high-stakes crime.

## C. Enforcement & Bounties

* Courthouse: Clear infamy by paying 100 $VBV per Wanted Level. Clears grant REHABILITATED.
* Bounty Board: Tracks Outlaws (Wanted 10+). Clean Hunters (Wanted <= 2) collect $VBV for victories.

------------------------------
## 6. INTELLIGENCE & COUNTER-INTELLIGENCE
The Arena's intelligence layer provides tools for both gathering information and protecting your organization.

*   **Cyber-Audit**: A tactical item that reveals a target club's current treasury average and crash status. Use it to identify vulnerable targets for heists or market manipulation.
*   **Cyber-Counter**: A hardware trap that, when active, identifies the player who used a Cyber-Audit against your club. It's a single-use defense that deactivates after revealing an auditor.
*   **Cyber-Lock**: A high-tier hardware trap that prevents all Cyber-Audits against your club for 24 hours, encrypting your treasury data from prying eyes.
*   **Ghost Protocol**: A tactical item that allows players to pay $VBV to temporarily scramble their signal, making them "Undetectable" on the Bounty Board for 30 minutes.
*   **Player Reporting**: A system allowing players to report malicious activity (cheating, harassment, exploits) directly to Arena Security. Reports are logged and administrators are notified in real-time.

------------------------------
## 6. TOURNAMENTS## A. Bracket Architecture
Automated system hosts structured 8-player or 16-player bracket events.

* Buy-in: Pay entry via $VBV or$AVoi. Elite ranked players receive free passes.
* Prize Pools: Total accumulated entry pools distribute to the Top 5 performers.
* Governor Tax: Arena Center territory owners claim 5% of gross tournament pools.
* DNF Penalties: Disconnects trigger severe, escalating Wanted Level and Reputation losses.
* Blockchain Ledger: Match history data writes permanently on-chain for verifiable validation.

------------------------------
## 7. SOCIAL STANDING & ACHIEVEMENTS## A. Reputation Dynamics
Your global social status scales dynamically based on live ecosystem inputs:

### Technical Note: Asset Standards
The Virtualbabes Oracle utilizes an intelligent **Metadata Dispatcher** to provide the broadest compatibility in the ecosystem:
1. **ARC-72**: High-performance smart-contract tokens.
2. **ARC-19**: Dynamic NFTs where metadata evolves via the Reserve Address.
3. **ARC-69**: Classic Algorand NFTs utilizing transaction notes.
4. **Cross-Chain**: Discover metadata from Ethereum, Polygon, and Solana via linked wallets.

➕ INCREASES STATUS              ➖ DECREASES STATUS
• Match Victories                • DNF / Disconnect Streaks
• Achievement Unlocks            • High Wanted Levels
• Premium Equipped Cosmetics     • Jailed Collection Cards
• Positive Rumor Density         • Target Negative Rumors

## B. Hall of Valor
Unlock unique profile badges, status boosts, and multipliers by hitting milestones:

* FIRST_VICTORY — Secure your initial Arena match win.
* EXECUTIVE_PAY — Sign an employment contract worth 500+ $VBV.
* ART_COLLECTOR — Win 3 separate Art Gallery bundle auctions.
* REHABILITATED — Fully wipe an active Wanted Level at the Courthouse.
