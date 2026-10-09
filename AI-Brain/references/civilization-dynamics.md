# Civilization Dynamics

> Deep reference for the AI civilization systems: justice, employment, rivalry, market, career progression, events, industrial loop, and the full behavioral tick model.

---

## BehavioralTick Architecture

Two distinct sections in `BehavioralTick`:

1. **Career switch** — career-dependent `execute*` called once per tick
2. **Probabilistic behavioral extensions** — each has its own `if citizen.Tier >= ...` guard and probability roll

**Integration Order:**
```
1. FREE_AGENT seek contract (continue)
2. Career-dependent execute* (switch)
3. Faith rituals (if DogmaTag)
4. Domestic (if Tier≥Journeyman)
5. Justice enforcement (if Tier≥Expert)
6. Employment (if EMPLOYED)
7. Rivalry (if Tier≥Journeyman)
8. Market activity (if Tier≥Expert)
9. Career progression (3% chance)
10. Event hosting (if Tier≥Master)
```

**Pitfall:** Adding a new behavior requires choosing the CORRECT section. Career actions go in the switch; lifecycle/justice/economy actions go after with their own guard. Mixing them breaks the probability model.

---

## 1. Justice Enforcement (`executeJusticeEnforcement`)

**Trigger:** Tier ≥ Expert, career ∈ {BountyHunter, Warden, ForensicAnalyst, JusticeRecruiter}

**Probability:** 7% per tick

**Logic:**
- Scans region for citizens with `WantedLevel ≥ 10`
- If found: captures citizen, adds bounty to treasury, reputation+
- If none found: generic justice action (XP+)

**Data flow:** `executeJusticeEnforcement` → `justiceService.CaptureCitizen()` → treasury delta → reputation delta

---

## 2. Employment (`executeEmployment`)

**Trigger:** Status == EMPLOYED

**Probability:** Every tick (not probabilistic)

**Logic:** Earn salary = `Tier × 50,000 micro` from owned club's treasury

**Data flow:** Club treasury → citizen wallet

---

## 3. Rivalry (`executeRivalry`)

**Trigger:** Tier ≥ Journeyman, career in antagonistic set

**Probability:** 4% per tick

**Logic:**
- Finds rival citizen (different pathway or faction)
- Reputation swing ±
- Treasury drain from losing side
- Optionally triggers duel

---

## 4. Market Activity (`executeMarketActivity`)

**Trigger:** Tier ≥ Expert

**Probability:** 10% per tick

**Logic:**
- If treasury ≥ threshold → invest in entity market
- If treasury < threshold or underworld → black market sale

**Pitfall:** All math is uint64 micro. Never convert to float for comparison.

---

## 5. Career Progression (`executeCareerProgression`)

**Trigger:** Any citizen

**Probability:** 3% per tick

**Logic:** Tier++ when XP + reputation exceeds threshold for next tier

**Tier Gate Mapping:**
| Tier | Name | Key Unlocks |
|---|---|---|
| 0 | Peon | Basic actions |
| 1 | Apprentice | Employment eligibility |
| 2 | Journeyman | Marriage, rivalry, pet adoption |
| 3 | Expert | Faith rituals, justice, market |
| 4 | Master | Event hosting |
| 5 | Boss | Full autonomy |

---

## 6. Event Hosting (`executeEventHosting`)

**Trigger:** Tier ≥ Master

**Probability:** 2.5% per tick

**Logic:** Hosts entity event for region → participants earn rewards

---

## 7. Domestic System

### Marriage (`executeMarriage`)
- Trigger: Tier ≥ Journeyman, unmarried
- Probability: 10% per tick
- Effect: `BondGraph.WifeWallet` set, `DomesticCoherence+`

### Breeding (`executeBreeding`)
- Trigger: Married + Certified + Expert+
- Probability: 5% per tick
- Effect: Spawns child `AICitizen` with inherited Certified status, `BotChildIDs+`

### Pet Adoption (`executePetAdoption`)
- Trigger: Tier ≥ Journeyman, <2 pets
- Probability: 8% per tick
- Effect: `PetIDs+`, `DomesticCoherence+`

**BondGraph Fields:** `WifeWallet`, `BotChildIDs`, `PetIDs` — auto-initialized on first domestic action.

**Coherence Formula:**
```
DomesticCoherence(region) = Σ_citizens (wife_count×15 + children_count×10 + pets_count×5)
```

---

## 8. Industrial Loop

**Circulation Engine:**
```
Player Activity → Businesses → Employment → Purchasing → Taxes → Treasuries → Development → Events → Player Activity
```

**Employment Service (`employment_service.go`):**
- Get hired at clubs (Manager, Security, Clerk)
- Salary from club treasury
- Club tiers unlocked with Mojo

**Club Management:**
- Set salaries
- Manage treasury
- Unlock tiers with Mojo (RequiredMojo gate)

**Taxation:**
- Auto-taxation routing
- Treasury management
- Governor can set district taxes (0-25%)

---

## Key Constraints

| Constraint | Value |
|---|---|
| Treasury math | uint64 micro-units (NO FLOAT) |
| Reputation range | [-100, 100] |
| Tier range | [0, 6] (Peon → Boss) |
| Region cap | 1 + regionIndex |
| AI wallet mandate | MUST hold own wallet (never personal) |
| Bot child inheritance | Certified from parents |

---

## HTTP Routes

| Route | Handler | Purpose |
|---|---|---|
| `/api/ai/citizens/spawn` | `handleSpawnAI` | Spawn new citizen with own wallet |
| `/api/ai/citizens/list` | inline | List all citizens |
| `/api/ai/citizens/stats` | inline | Aggregate stats |
| `/api/ai/citizens/free-agents` | inline | List FREE_AGENT citizens |
| `/api/ai/citizens/marry` | `handleMarryAI` | Execute marriage |
| `/api/ai/citizens/breed` | `handleBreedAI` | Execute breeding |
| `/api/ai/citizens/adopt-pet` | `handleAdoptPetAI` | Execute pet adoption |
| `/api/ai/citizens/progress` | `handleAIProgression` | Execute career progression |

---

## Dev Seed

2 AI citizens spawn on boot for testing. They begin at Tier 0 (Peon) and follow BehavioralTick each cycle.

---

## Matrix Dynamics & Cross-System Effects

| Dynamic | Feeds Into | How |
|---|---|---|
| Justice Enforcement | Reputation, Treasury | Capture → bounty → treasury+ |
| Employment | Economy | Salary circulation from club |
| Rivalry | SocialRank, Theme | Reputation swing → theme vector |
| Market Activity | MarketVitality | Investment → entity market depth |
| Career Progression | All systems | Tier unlocks new behaviors |
| Event Hosting | EventDynamics | Events → region vitality |
| Domestic | DomesticCoherence | Marriage/breeding/pets → BondGraph |
| Faith Rituals | FaithCoherence | Rituals → Awe → world gravity |

---

## Pitfall Reference

1. **Career vs lifecycle placement:** Career actions go in the switch; everything else goes after. Mixing = broken probability.
2. **Uint64 micro math:** Never use float64 for treasury, salary, or investment amounts.
3. **BondGraph auto-init:** Don't check for nil before accessing — auto-initialize on first domestic action.
4. **Bot child Certified:** Must inherit from parents, or black-market ineligibility applies.
5. **Region cap:** `1 + regionIndex` — adding citizens beyond this silently fails.
