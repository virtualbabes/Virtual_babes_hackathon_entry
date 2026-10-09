// ============================================================================
// collective-intelligence.js — RE-EXPORT ALIAS (NOT an implementation)
// ----------------------------------------------------------------------------
// ONE implementation exists: `Public/collective-intelligence.js` (the narrative
// registry + `generatePlaystyleTaunt`), which is what `app.js`, `economy.js` and
// `game.js` already load.
//
// WHY AN ALIAS AND NOT A SECOND MODULE. This path used to hold a separate,
// EMPTY registry (`personalities: {}`, no `generatePlaystyleTaunt`). Nothing
// imported it — because every consumer reaches the root file, either directly
// (`app.js` → `./collective-intelligence.js`) or by escaping one level up
// (`Public/js/economy.js` → `../collective-intelligence.js`). That made it a
// TRAP: a future in-directory import (`./collective-intelligence.js` from
// `Public/js/`) would have resolved here and silently produced an empty
// personality registry — an NPC layer that still "worked" while knowing
// nothing. Two same-named modules with different content is one owner too many,
// so this file now forwards to the single implementation and defines no state
// of its own (`npm run verify:duplicates` enforces exactly this rule).
// ============================================================================

export { collectiveIntelligence } from '../collective-intelligence.js';
