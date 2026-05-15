# Virtualbabes Arena: Player's User Manual

Welcome to the Virtualbabes Arena, a high-stakes Social Economic Simulation where tactical card combat meets deep organizational management, high-finance markets, and the thrilling underworld of criminality. This manual will guide you through the core mechanics and features of the Arena.

---

## 1. Getting Started

### A. Connecting Your Wallet
To enter the Arena, you'll need to connect your Voi wallet. The Arena supports:
*   **Nautilus Wallet**
*   **Kibisis Wallet**
*   **WalletConnect** (for other compatible wallets)

Your wallet address will be your unique identifier in the Arena.

### B. Choosing Your Avatar
After connecting, you'll select an avatar from your owned NFTs. This avatar represents you in the lobby and during matches. You can also set a "Gloat Message" – a short phrase displayed if you win a match.

### C. Building Your Deck
Your deck consists of 5 cards. You can manage your cards in the **Deck Manager**:
*   **Inventory:** All cards you own.
*   **Deck Slots:** You have multiple deck slots. Higher reputation unlocks more slots.
*   **Auto-Build:** The system can automatically build a deck for you based on optimal power.

---

## 2. Combat & Matchmaking

### A. Matchmaking
Once your deck is ready, you can join the **Matchmaking Pool** to find an opponent. The system will try to pair you based on your Reputation and Deck Rating.

### B. Combat Rules (Triple Triad-inspired)
The game is played on a 3x3 grid. Each card has 4 power values (Top, Right, Bottom, Left).
*   **Basic Capture:** Place a card next to an opponent's card. If your adjacent power is higher, you capture it.
*   **Same Rule:** If two or more of your card's adjacent sides match the power of an opponent's card, you capture them all.
*   **Plus Rule:** If two or more of your card's adjacent sides *sum* to the same value as an opponent's card, you capture them all.
*   **Combo Chain:** Capturing a card can trigger further captures if its newly acquired sides are stronger than adjacent opponent cards.
*   **Elemental Synthesis:** Cards have Moods (Volatile, Serene, Spirited, Grounded). Tiles on the board can also have Moods. Matching Moods grant a power bonus; opposing Moods incur a penalty.
*   **Fallen Penalty:** Cards captured during a match suffer a permanent power reduction.
*   **Sudden Death:** If a match ends in a 5-5 draw, a Sudden Death round is triggered. The board is cleared, and remaining cards are redistributed based on current ownership for a tie-breaker.

### C. Player Attributes
Your performance and actions influence your attributes:
*   **Mojo:** Your social standing and influence. Higher Mojo unlocks better items and boosts your employees' reputation.
*   **Reputation:** Your overall standing in the Arena. Influences your share price in the Entity Market.
*   **Wanted Level:** Increases with criminal activity (Heists, Black Market purchases). High Wanted Levels incur power penalties in combat.
*   **Cunning:** Improves your chances of success in criminal activities and mitigates Wanted Level penalties.
*   **Nurturing:** Reduces the Fatigue penalty on your cards.

---

## 3. The Industrial Loop (Clubs & Economy)

### A. Clubs & Territories
*   **Founding a Club:** For a fee, you can found your own Club and claim an unclaimed territory.
*   **Joining a Club:** You can join an existing Club for a fee.
*   **Territories:** Clubs own territories. Owning 2+ territories makes your Club a **Region**, and you become a **Regional Governor**.
*   **Regional Power Boost:** Members of a Governor's Club receive a +5% power bonus in matches played within their controlled districts.
*   **Mojo:** Clubs earn Mojo through shop turnover and successful heist defenses. High Mojo unlocks specialized items.

### B. Employment & Careers
Club Owners can hire other players into roles:
*   **Manager:** Manages shop commission rates and inventory.
*   **Security:** Reduces heist success chance against the club, deploys traps.
*   **Clerk:** Increases shop turnover speed.

Being employed boosts your Reputation. High-value contracts (500+ $VBV) earn the **EXECUTIVE_PAY** achievement.

### C. Shops & Items
Clubs operate shops in their territories, selling tactical items:
*   **Elemental Forge:** Mood Catalysts, Grounded Shields, Prism Shields.
*   **Tactical Syndicate:** Rule Breakers, Intel Reports, Ghost Protocol.
*   **Vitality Lab:** Stamina Stims, Loyalty Pledges, Hyper-Stims.
*   **Hardware/Security:** Laser Tripwires, Sentry Turrets, Bio-Guard Dogs.

Some items are **Master Tier**, requiring your Club to be a Regional Governor to purchase. Others are **Role-Gated**, requiring a specific job role (e.g., Security for traps).

### D. Industrial Leases
Club members can list their cards for rent on the **Lease Board**.
*   **Revenue Split:** Lease payments are distributed: 50% to the Lender, 20% to the Faucet (Arena Tax), 20% to the Club Treasury, and 10% to Club Members.

---

## 4. High-Finance & Markets

### A. Entity Market
Trade shares in other players and NPCs. Share prices are dynamic, influenced by Reputation, Wins, and Rumors.

### B. Art Gallery (Auctions)
List unique **Card Bundles** (Card + Weapon + Faceplate) for auction. The Gallery (or the Club controlling "the_art_gallery" territory) takes a 10% commission. Winning 3+ auctions earns the **ART_COLLECTOR** achievement.

### C. Second-Hand Store (Loans)
Use your cards as collateral to get immediate $VBV liquidity. Defaulting on a loan liquidates your card into **Market Tokens** and moves it to the Black Market.

### D. Black Market
Access this restricted market (requires Wanted Level 5+ and Cunning 10+) to buy liquidated collateral from defaulted loans at a discount. Purchases increase your Wanted Level.

### E. Rumor Mill
Pay to spread positive or negative rumors about other players or NPCs. Rumors temporarily influence their share price in the Entity Market. Regional Governors receive a portion of rumor fees.

---

## 5. Criminality & Justice

### A. Heists
Attempt to loot other Club Treasuries. Success chance depends on your Cunning vs. the target's Security Level (Mojo, Security Staff, deployed Traps).
*   **Fence Fee:** A 10% fee on successful loot is returned to the Faucet.
*   **Guard Dog:** If a target Club has a Bio-Guard Dog, a failed heist can result in your rarest card being jailed.

### B. Jailing
If your card is jailed (by a Guard Dog or Fallen Penalty), it's held by the capturing Club.
*   **Bail:** You can pay a 200 $VBV fine to the jailing Club to release your card.
*   **Reputation Penalty:** Jailed cards reduce your Reputation.

### C. Kidnap Gambit
Successful high-stakes heists can lead to a **Kidnap Gambit**. You can "hold hostage" an opponent's favorite or rarest card.
*   **Ransom:** The victim can pay a $VBV ransom to reclaim their card. A 20% "Laundering Tax" is returned to the Faucet.
*   **Insurance Recovery:** If no ransom is paid, the card is automatically returned to the victim after 48 hours.

### D. Courthouse
Pay a fine (100 $VBV per Wanted Level point) to clear your **Wanted Level**. Fines are distributed to the Faucet and active Club Treasuries. Paying your fine earns the **REHABILITATED** achievement.

### E. Bounty Board
Track high-infamy "Outlaws" (Wanted Level 10+) currently in the lobby. Hunters (Wanted Level <= 2) can challenge and earn scaled $VBV bounties for victories.

---

## 6. Tournaments

### A. Automated Events
Participate in 8 or 16-player bracket tournaments.
*   **Buy-in:** Pay a $VBV or $AVoi buy-in to enter. Top-ranked players ("Elite Privilege") may receive free passes.
*   **Prize Pool:** The total pot (including buy-ins and bonuses) is distributed to the Top 5 finishers.
*   **Governor's Tax:** 5% of the total pot is routed to the Club controlling the "Arena Center" territory.
*   **Payouts:** Rewards are distributed based on rank, with reputation-based tie-breakers.
*   **DNF Penalties:** Disconnecting from a tournament match incurs escalating Wanted Level and Reputation penalties based on the round.
*   **On-Chain Archiving:** Tournament results are permanently recorded on the blockchain for verifiable history.

---

## 7. Social Standing & Achievements

### A. Reputation & Social Rank
Your Reputation (Standing) is a key metric, influencing your share price in the Entity Market. It's affected by:
*   Wins, DNFs, Disconnect Streaks
*   Wanted Level (penalty)
*   Jailed Cards (penalty)
*   Achievements (bonuses)
*   Playstyle (Aggressiveness, Risk Tolerance)
*   Employment (multiplier from Club Mojo)
*   Cosmetics (Faceplate bonuses)
*   Rumor Count (bonus for social influence)

### B. Hall of Valor (Achievements)
Earn trophies for various milestones (e.g., **FIRST_VICTORY**, **TOURNAMENT_CHAMPION**, **FIRST_HEIST**, **ART_COLLECTOR**, **GOVERNOR**). Achievements boost your Reputation.

### C. NPC Intelligence
NPCs observe your playstyle (Aggressiveness, Risk Tolerance, Preferred Rules) and will generate contextual taunts in the lobby or during matches.

---

## 8. Technical Notes

*   **Blockchain Integration:** The Arena uses Voi and Algorand for transactions and state verification. It also discovers NFTs from other chains (Ethereum, Solana, Polygon, etc.) to integrate them into your card collection.
*   **Deterministic Gameplay:** The game's core rules run on a WebAssembly (WASM) engine in your browser, ensuring fair and consistent gameplay.
*   **Security:** Your private keys are never exposed to the server. All critical transactions are verified on-chain.

---

Enjoy your time in the Virtualbabes Arena! May your strategies be sharp and your reputation soar.