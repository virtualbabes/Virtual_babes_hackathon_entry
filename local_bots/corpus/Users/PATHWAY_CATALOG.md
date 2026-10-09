# PATHWAY CATALOG — Users / Bots / Pets

> Master index of every local-LLM pathway corpus in `local_bots/corpus/`.
> Each entry maps a pathway → its corpus dir → the handbook it was curated from → its eligibility gate.
> Themed like the old user manuals; this catalog is the single source for bot/pet voice provisioning.

## USERS (player-facing)
| Pathway | Corpus dir | Source handbook | Notes |
|---------|-----------|-----------------|-------|
| citizen_help | `Users/citizen_help` | NEW `User_manual` (this tree) | Canonical player guide + CITIZEN bot source |

## BOTS (AI-citizen voices — §24.5/§24.6 promotion)
| Pathway | Corpus dir | Source handbook | Eligibility gate |
|---------|-----------|-----------------|-----------------|
| citizen | `Bots/citizen` | User_manual + Canonical Flow Bible | Certified bot-child, L≥25 |
| career | `Bots/career` | Professional Career Pathways | Certified bot-child, L≥25 |
| governor | `Bots/governor` | Regional Governance Strategy | Certified bot-child, L≥25 |
| sovereign | `Bots/sovereign` | Sovereign's Blueprint + Voi Purist | Certified bot-child, L≥25 |
| underworld | `Bots/underworld` | Underworld Extraction | Certified bot-child, L≥25 (black-market INELIGIBLE) |
| justice | `Bots/justice` | JUSTICE SHALL PREVAIL | Certified bot-child, L≥25 |
| breeder | `Bots/breeder` | Mutation Foundry | Certified bot-child, L≥25 |
| family | `Bots/family` | NEW (no handbook) | Certified bot-child, L≥25; voices domestic bonds |

## PETS (special local-LLM tier — Pet World §30 stats LIVE; tournament payout deferred)
| Pathway | Corpus dir | Source | Eligibility gate |
|---------|-----------|-------|------------------|
| pet_generic | `Pets/pet_generic` | NEW (themed manual) | Certified PetNFT, mature |
| pet_immature | `Pets/pet_immature` | NEW (themed manual) | Certified PetNFT, pre-maturity (learning tier) |

> Pet stats (Speed/Intelligence/Willpower/Strength/Charisma/Agility, uint64) are LIVE via §30
> `EntityStats` + 3D power overlay. Tournament reward payout reuses `TournamentService` (future).

## SHARED ELIGIBILITY (constitutional)
- Certified == true AND BlackMarketAdopted == false (legitimacy = right to think).
- Children-bots, immature pets, and other bots are ALL eligible for their respective tier.
- All earn rewards in world-changing events (hook contract: `/api/events/world-shaking`, uint64 micro,
  implementation deferred to the Pet World plan).

## BUILD PATH
1. Curate a pathway's `README.md` (this tree).
2. Run `setup_bot_pathway.bat <pathway>` (wrapper; feeds corpus → existing ornith harness).
3. Harness emits a per-pathway runner into the owner's Ollama dir.
4. `local_model_promotion.go` TriggerBuild calls the wrapper when `VB_MODEL_COMPILE_ENABLED=true`.
