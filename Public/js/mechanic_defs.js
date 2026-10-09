// mechanic_defs.js — Example mechanic definitions (§25.5 developer game-hub)
// These are CONCRETE, working examples of the Mechanic framework. Developers add more
// by calling engine.register({...}) at runtime. All effects are uint64 micro-units.
//
// WHAT THESE MIRROR — AND WHAT "MIRROR" HONESTLY MEANS HERE. Every constant below is a HAND-COPIED
// restatement of a server-side number, and **NOTHING IN THE TREE COMPARES THE TWO SIDES**:
//   - region cap = 1 + regionIndex   (ai_citizen_engine.go SpawnAI)   — `cap` is passed in ctx.meta and
//                                                                        currently READ BY NO compute below
//   - club mojo / treasury            (club_service.go)               — `/100n`
//   - rivalry weights W_*             (theme_engine.go const block)   — not restated here at all
//   - theme gravity                   (§27.3 AI-Citizen Population Gravity) — `gravityMicro` is called a
//                                                                        "future server field" by the
//                                                                        definition itself, so it is 0 today
// THE CLAIM THAT USED TO STAND HERE — that these mirrors make "the 3D world's emergent numbers agree
// with the authoritative simulation" — was PROSE ONLY and has been corrected rather than left standing:
// the agreement is asserted by nobody. It becomes real when a test compares these literals with the Go
// constants that own them; until then, treat a client-side number as a UI-derived estimate and the
// server as the authority.

import { Mechanic } from './mechanics.js';

// ---- Region vitality: more AI citizens + caches => richer region (stackable) ----
export const M_REGION_VITALITY = new Mechanic({
  id: 'region_vitality',
  kind: 'vitality',
  scope: 'region',
  weight: 1,
  repeatable: true,
  stackable: true,
  rivalryTag: 'region_vitality',
  // ctx.meta: { citizenCount, cacheCount, cap }
  compute: (ctx) => {
    const citizens = BigInt(ctx.meta.citizenCount || 0);
    const caches = BigInt(ctx.meta.cacheCount || 0);
    // 50_000 micro per citizen + 120_000 per live cache (integer)
    return citizens * 50000n + caches * 120000n;
  },
});

// ---- Club mojo field: mojo boosts regional coherence (stackable, repeatable per tick) ----
export const M_CLUB_MOJO = new Mechanic({
  id: 'club_mojo',
  kind: 'mojo',
  scope: 'region',
  weight: 1,
  repeatable: true,
  stackable: true,
  rivalryTag: 'mojo_field',
  // ctx.meta: { mojoMicro }  (uint64 micro from server club treasury/mojo)
  compute: (ctx) => {
    const mojo = BigInt(ctx.meta.mojoMicro || 0);
    // 1% of mojo bleeds into region coherence (integer divide)
    return (mojo / 100n) > 0n ? (mojo / 100n) : 0n;
  },
});

// ---- Rivalry aura: contested regions emit a rivalry field (once-per-resolve, stackable) ----
export const M_RIVALRY_AURA = new Mechanic({
  id: 'rivalry_aura',
  kind: 'aura',
  scope: 'region',
  weight: 1,
  repeatable: false, // one aura field per region per resolve
  stackable: true,
  rivalryTag: 'rivalry_aura',
  // ctx.meta: { contestedMicro }  from server rivalry engine
  compute: (ctx) => BigInt(ctx.meta.contestedMicro || 0),
});

// ---- Theme gravity: AI-citizen population gravity pulls players (§27.3) ----
export const M_THEME_GRAVITY = new Mechanic({
  id: 'theme_gravity',
  kind: 'gravity',
  scope: 'region',
  weight: 1,
  repeatable: true,
  stackable: true,
  rivalryTag: null,
  // ctx.meta: { gravityMicro }  serialized in /api/regions (future server field)
  compute: (ctx) => BigInt(ctx.meta.gravityMicro || 0),
});

// ---- Player career XP ripple: a player's career progress emits local vitality ----
export const M_CAREER_RIPPLE = new Mechanic({
  id: 'career_ripple',
  kind: 'vitality',
  scope: 'player',
  weight: 1,
  repeatable: true,
  stackable: true,
  rivalryTag: 'career_ripple',
  // ctx.meta: { careerXPMicro }
  compute: (ctx) => BigInt(ctx.meta.careerXPMicro || 0) / 10n,
});

export const DEFAULT_MECHANICS = [
  M_REGION_VITALITY,
  M_CLUB_MOJO,
  M_RIVALRY_AURA,
  M_THEME_GRAVITY,
  M_CAREER_RIPPLE,
];

export default DEFAULT_MECHANICS;
