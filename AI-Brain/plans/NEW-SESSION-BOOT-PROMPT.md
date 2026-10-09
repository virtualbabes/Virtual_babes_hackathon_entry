# NEW-SESSION BOOT PROMPT — copy everything between the markers

---

**▼ COPY FROM HERE ▼**

Continue the NFT-Seduction build. Read these first, in order: `.clinerules/Session-Handoff.md` (**v41.0 — read the "REMAINING WORK — THE COMPLETE LIST"**), `AI-Brain/plans/VOI-TENANT-FRAMEWORK-PLAN.md` (**14 sections — the ONE owner of the vision and of the record contract**), and `.clinerules/active_directive.md`.

**THE MODEL (settled, do not re-litigate):** Voi is the sole base chain; every other chain is a plug-in · payouts are **$VBV on Voi only** · every token other than $VBV is **INWARD-ONLY** ($VBV + $VOI fallback on Voi; $AVOI + $ALGO fallback on Algorand; $NUGGET/$UNIT on both) · **all records are $VBV transaction notes** — a note on the **ARC-200 transaction from the faucet vault**, **NOT an app-call** — gzipped, compared against the local cache, which exists **only for runtime smoothness** · **the seed IS the record/save file and must restore the ENTIRE server state**, possibly via **sequential gzip pulls** · an Algorand user never needs Voi · `Nugget`/`Unit` shops are **universal sellers** transacting only in their own token, inward-only · dev-hub tenants lease the architecture, **pay the toll in $VBV**, seed their own records with a **vault-only** address given to **their own bot**, and **the app holds no tenant key, ever** · the dev/game hub and the full lease catalogue come **LAST**.

**SCOPE OF THIS SESSION:** work **sequentially** through the REMAINING WORK list (A1 → A2 → A3 → A4 → A5 → A6, then B, then C, then D), starting at **A1 — declare the 13 remaining primary families**. Each family is a four-edit pattern: vocabulary constant → `noteVocabulary` catalogue entry → the family's `Prefix` in `record_families.go` → advance the census numbers in `record_families_test.go`. **The census is the progress meter (9 → 22 declared).**

**RULES THAT ARE BINDING:**
- **Do not duplicate code or a call to a transaction.** A record rides on the transaction that already moves the value; only a record with no movement of its own is a single 1-micro self-transfer. One owner per concern; extend existing owners.
- `RECORDS_DISPATCH` stays **OFF**; **no live/on-chain testing until the UI is complete**; **the seed is planted only after the UI**.
- Secrets are env-only; nothing seeds while any primary fact is PENDING; never guess an asset id ($NUGGET/$UNIT ids come from Brendan).
- **Verify before reporting:** native + `linux/amd64` + `js/wasm` builds rc 0; `go test .` green except the PRE-EXISTING `TestCalculateBuyCost_WhaleSlippagePenalty`; `Public/main.wasm` untouched (11,375,951 B, mtime 09/16 11:50); no NEW gofmt drift; never run `gofmt -w` repo-wide; `git status` shows only intended files.
- **GIT PUSH is Brendan's** — never attempt it.
- **Compress context when it gets tight:** update `.clinerules/Session-Handoff.md` and the plan file with the current state and the exact next step, then continue.

**REPORTING:** for each unit of work, state what changed, the measured evidence, and the next step; end with recommendations.

**▲ TO HERE ▲**