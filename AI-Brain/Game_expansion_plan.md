# Virtualbabes Arena: Game Expansion Plan

## 1. Vision Statement
To evolve Virtualbabes Arena from a tactical card battler into a high-stakes **Social Economic Simulation**. The game will reward not just combat skill, but also strategic investment, political maneuvering within Card Clubs, and the management of one's "Social Standing" (Reputation and Mojo).

---

## 2. Current Ecosystem Audit (Beta State)
| Feature | Status | Description |
| :--- | :--- | :--- |
| **Combat Engine** | Complete | WASM-based rules (Same, Plus, Combo) with authoritative server validation. |
| **Infamy System** | Complete | Wanted Levels, Heists, and Courthouse fines (100 $VBV/point). |
| **Entity Market** | Complete | Stock trading of player/NPC shares using internal $VBV reward balances. |
| **Clubs/Territories** | Functional | Club founding, joining, and basic revenue loops (Shop turnover/Kickbacks). |
| **Achievements** | Functional | Persistent trophy system (Outlaw Slayer, Perfect Game, etc.). |
| **X-Chain Oracle** | Complete | Multi-chain NFT discovery (AVM, EVM, Solana) with power-scaling. |

---

## 3. Pillar 1: The Industrial & Trust Layer
**Goal:** Create a social hierarchy through employment, property, and regional dominance.

### A. In-Game Employment & Careers
*   **Roles:** Club Owners can hire other players into specialized roles:
    *   **Manager:** Can adjust commission rates and restock shop inventory.
    *   **Security:** Reduces the success chance of Heists. Can manage **Traps** (Tripwires, Sentries) and **Guard Dogs** purchased from the Hardware Store.
    *   **Clerk:** Increases shop turnover speed and earns a small base salary from the treasury.
*   **Trust:** Employment creates a "Service Record" in `PlayerStats`, making reliable employees highly valuable in the market.

### B. Courthouse Rerouting (Implemented)
*   **Mechanism:** $VBV fines paid at the Courthouse are no longer burned.
*   **Logic:** 50% returns to the Faucet pool; 50% is distributed equally among active Club Treasuries.
*   **Narrative:** Clubs act as the "Security Guilds" of the Arena.

### B. Club Mojo & Tiered Unlocks
*   **Mechanism:** Clubs earn "Mojo" through successful tournament placements of members and high shop turnover.
*   **Unlock:** High-Mojo clubs unlock specialized items in their shops (e.g., Rare Mood Catalysts, Anti-Fatigue Stims).

### C. Regional Expansion
*   **Mechanism:** Once a Club (or alliance) owns **2 Territories**, they form a **Region**.
*   **Buff:** Regions grant a global +5% power boost to all members within that district and unlock "Master" tier items.
*   **Governor:** The Club Owner becomes a Regional Governor, earning a small tax from all Courthouse fines paid by players caught within their region.

---

## 4. Pillar 2: The High-Finance & Market Layer
**Goal:** Port the deep economic loops from the build resources to the server.

### A. Art Gallery: Auctions & Consignments
*   **Mechanism:** Players can list specialized **Card Bundles** (Card + Weapon + Faceplate) for $VBV auction.
*   **Cut:** The Gallery (or the Club controlling that district) takes a 10% commission.

### B. Second-Hand Store: Loans & Collateral
*   **Mechanism:** Players can get immediate $VBV liquidity by using "Soul-Bonded" cards as collateral.
*   **Risk:** Failing to repay the loan results in the card being liquidated into **Market Tokens**, effectively "burning" the card but increasing the equity pool for that entity.
*   **Underworld:** High-cunning players can buy these defaulted cards from the "Black Market" at a discount, but they carry a "Stolen" tag that increases Wanted Level while held.

### C. Rumor Mill & Market Manipulation
*   **Mechanism:** Players can pay "The Salon" or "Hot Spot" managers to spread rumors.
*   **Effect:** Positive/Negative rumors apply multipliers to an entity's Share Price for a limited time.

---

## 5. Pillar 3: Criminality & Intelligence
**Goal:** Expand high-risk/high-reward gameplay.

### A. Kidnapping & Ransom
*   **Mechanism:** Successful high-stakes heists can lead to a **Kidnap Gambit**.
*   **Logic:** A player can "hold hostage" an NPC's (or another player's) favorite card. The victim must pay a $VBV ransom or wait for an "Insurance Recovery" cycle.

### B. Collective NPC Intelligence
*   **Mechanism:** A server-side observation loop (ported from `collective-consciousness.js`).
*   **Narrative:** NPCs recognize your playstyle. If you always use "Plus" rules, they will taunt you about it in the Lobby or during matches.

---

## 6. Pillar 4: Performative Market & Social Flex
**Goal:** Turn player performance into a liquid asset.



### A. Enhanced Portfolio View
*   **UI Update:** Display Achievement Badges (Trophies) next to player names in the Market Ticker and Portfolio list.
*   **Valuation Logic:** Achievement counts should act as a multiplier for Share Price, rewarding "Decorated Veterans."

### B. Social Sharing (X/Twitter)
*   **Mechanism:** One-click sharing of Match Results, Heist Successes, and Trophy Unlocks to drive external growth.

---

## 5. Pillar 3: Deep RPG Mechanics
**Goal:** Increase card usage strategy via persistence.

### A. The Fatigue/Loyalty Loop
*   **Fatigue:** Overused cards lose power (-1 per match above 50 usage).
*   **Loyalty:** Soul-bonded cards gain power (+25 at max loyalty).
*   **Sink:** "Vitality Lab" Club shops sell consumables to manage these stats.

### B. Elemental Synthesis
*   **Mechanism:** Aligning card Moods with Tile Moods for significant power boosts (+50).
*   **Sink:** "Elemental Forge" Club shops sell Mood-alignment artifacts.

---

## 6. Technical Roadmap

### Phase 1: Commercialization (Next)
*   Implement specialized inventory and Staffing slots for the three Club Types.
*   Refine `CalculateReputation` to include Mojo and Achievement counts.

### Phase 2: Seasonality
*   Implement automated "Season Rollover" in `lobby_manager.go`.
*   Persistent Hall of Fame (HoF) archives recorded on-chain via transaction notes.

### Phase 3: Advanced Socials
*   Global Lobby Chat filters and Admin moderation tools.
*   "Bounty Board" UI to visualize high-Wanted players currently in the matchmaking queue.

---

## 7. Economic Guardrails
*   **Anti-Inflation:** All $VBV remains within the loop (Faucet -> Players -> Clubs -> Shops -> Faucet).
*   **Sybil Protection:** Continue utilizing historical Indexer checks for Bridge/Onboarding rewards.
*   **Governance:** Club Managers set commission rates within a regulated 5-50% window.


*********************************************************************

Dev Summary:
The Virtualbabes Arena is rapidly evolving from a tactical card battler into a sophisticated Social Economic Simulation. The expansion flow is structured around several interconnected pillars, each designed to deepen player engagement, create dynamic economic loops, and foster a vibrant in-game society.

Here's a breakdown of the expansion flow and what we should focus on next:

The Grand Expansion Flow: Pillars of the Arena
Pillar 1: The Industrial & Trust Layer

Goal: Establish a social hierarchy through employment, property ownership, and regional dominance.
Flow: Players found Clubs, acquire Territories, and hire other players into specialized roles (Manager, Security, Clerk). This creates a "Service Record" for players, building social trust. Clubs earn revenue from shop commissions and tournament kickbacks.
Current Status:
In-Game Employment & Careers: Implemented. Club owners can hire players, assign roles, and set salaries. A background ticker dispenses daily salaries from Club Treasuries.
Courthouse Rerouting: Implemented. $VBV fines are now distributed to active Club Treasuries, making Clubs the "Security Guilds."
Club Mojo & Tiered Unlocks: Mojo exists, but its direct impact on tiered shop unlocks is pending.
Regional Expansion: Implemented. Clubs owning 2+ territories become "Regional Governors," gaining a 15% tax on Courthouse fines from their region. Territory acquisition is also implemented.
Pillar 2: The High-Finance & Market Layer

Goal: Port deep economic loops from the build resources to the server, creating a liquid, player-driven market.
Flow: Players can auction unique card bundles, take collateralized loans against their NFTs, and engage in market manipulation. Defaulted collateral is liquidated into "Market Tokens" and sold on the Black Market.
Current Status:
Art Gallery: Auctions & Consignments: Implemented. Players can list card bundles for $VBV auction, with Club commissions.
Second-Hand Store: Loans & Collateral: Implemented. Players can take $VBV loans using NFT card bundles as collateral. Automated default processing is in place.
Market Token Liquidation & Black Market: Implemented. Defaulted collateral is moved to the Black Market, and borrowers receive Market Tokens. The Black Market UI is also implemented, gated by Wanted Level and Cunning.
Rumor Mill & Market Manipulation: Pending.
Pillar 3: Criminality & Intelligence

Goal: Expand high-risk/high-reward gameplay, adding layers of intrigue and consequence.
Flow: Players can engage in Kidnapping, face Ransom demands, and interact with NPCs who "remember" their playstyle.
Current Status:
Kidnapping & Ransom: Implemented. Hostage cards, ransom payments, and insurance recovery cycles are functional.
Collective NPC Intelligence: Implemented. Personality-driven taunts based on observed playstyle (aggressiveness/risk) are active.
Pillar 4: Performative Market & Social Flex

Goal: Turn player performance into a liquid asset and enable social showcasing.
Flow: Player achievements and reputation influence their market value. Social sharing tools allow players to broadcast their exploits.
Current Status:
Enhanced Portfolio View: Implemented. Market holdings, valuation, and share trading are functional.
Social Sharing (X/Twitter): Implemented. One-click victory sharing to X is functional.
Deep RPG Mechanics

Goal: Increase card usage strategy via persistence and elemental interactions.
Flow: Cards gain/lose power based on Fatigue and Loyalty. Elemental Moods on the board interact with card moods for tactical advantages.
Current Status:
The Fatigue/Loyalty Loop: Implemented. Items exist to manage these stats.
Elemental Synthesis: Implemented. Item buffs (Mood Catalyst, Grounded Shield) are applied and enforced in battle_service.go.
