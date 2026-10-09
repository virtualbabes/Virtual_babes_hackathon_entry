// mechanics.js — Composable Mechanic Framework (§25.5 / developer game-hub)
// PILLAR: deterministic, repeatable, stackable, rivalry-capable mechanics.
// CONSTITUTIONAL: ledger math is uint64 MICRO-UNITS ONLY. NO float/double in any
// economic or scoring value. Floats are permitted ONLY for transient view tints.
//
// Design goals (Brendan):
//   - simple but extensive: a mechanic is a tiny declarative object
//   - repeatable: same mechanic can fire many times (cooldown/once-per-scope guards)
//   - stackable: multiple mechanics on one scope SUM their integer effects
//   - rivalry-capable: any mechanic may carry a rivalryTag that feeds the rivalry matrix
//   - developer-extensible: register new mechanics at runtime; engine is data-driven
//
// This module is framework-agnostic (no Three.js import) so it can be unit-tested
// and reused by both the 2D lobby and the 3D world client.

const MICRO = 1_000_000n; // 1 unit = 1_000_000 micro-units (uint64 space)

// ---- deterministic helpers (integer only) ----
export function toMicro(units) {
  // Accept integer units or already-micro bigint. Never float.
  if (typeof units === 'bigint') return units;
  if (Number.isInteger(units)) return BigInt(units) * MICRO;
  throw new Error('toMicro: non-integer value violates uint64 ledger rule: ' + units);
}

export function addMicro(a, b) {
  return (typeof a === 'bigint' ? a : toMicro(a)) + (typeof b === 'bigint' ? b : toMicro(b));
}

// Stable integer hash for deterministic selection (no Math.random in ledger paths).
//
// THE PARENTHESES ARE LOAD-BEARING. `??` binds LOOSER than `<`, so the unparenthesised form
// `i < seedStr?.length ?? 0` evaluates as `(i < seedStr.length) ?? 0` — a BOOLEAN, which coerces to
// 1 or 0. The loop therefore ran for AT MOST ONE character, and returned the bare FNV offset basis for
// an empty string, so the helper documented as the stable deterministic hash COLLIDED across every id
// sharing a first character (the same `??`-precedence class the repository found in eight other client
// modules — see Problems.md). Shipped impact, measured: none of the five definitions in
// `mechanic_defs.js` reads `ctx.seed` (they all read `ctx.meta.*`), so no shipped number changed —
// but the contract a developer registers against was broken.
export function hashU64(seedStr) {
  let h = 14695981039346656037n; // FNV-1a 64-bit offset
  for (let i = 0; i < (seedStr?.length ?? 0); i++) {
    h ^= BigInt(seedStr.charCodeAt(i));
    h = (h * 1099511628211n) & 0xffffffffffffffffn;
  }
  return h;
}

// ---- Mechanic definition ----
// A Mechanic:
// {
//   id:        string            unique key
//   kind:      'vitality'|'mojo'|'aura'|'gravity'|'tax'|'custom'
//   scope:     'region'|'club'|'player'|'world'   where it applies
//   weight:    uint (integer multiplier, default 1) — stacks linearly
//   repeatable:boolean          can fire more than once on a scope
//   stackable: boolean          effects sum (vs replace)
//   rivalryTag: string|null      if set, contributes to rivalry matrix under this tag
//   oncePer:   string|null       dedupe key (e.g. 'per-day') — engine tracks fired set
//   compute:   (ctx) => bigint   PURE integer micro-delta. ctx has {seed, scopeState, meta}
// }
export class Mechanic {
  constructor(def) {
    if (!def || !def.id) throw new Error('Mechanic requires id');
    this.id = def.id;
    this.kind = def.kind || 'custom';
    this.scope = def.scope || 'world';
    this.weight = Number.isInteger(def.weight) ? BigInt(def.weight) : 1n;
    this.repeatable = def.repeatable !== false; // default true
    this.stackable = def.stackable !== false;   // default true
    this.rivalryTag = def.rivalryTag || null;
    this.oncePer = def.oncePer || null;
    this.compute = def.compute; // (ctx) => bigint (micro)
    if (typeof this.compute !== 'function') throw new Error('Mechanic ' + def.id + ' needs compute()');
  }
}

// ---- Mechanic Engine: stacks + resolves + rivalry feed ----
export class MechanicEngine {
  constructor() {
    this.mechanics = new Map();      // id -> Mechanic
    this.firedOnce = new Map();      // oncePer key -> Set(scopeKey)
    this.rivalry = new Map();        // rivalryTag -> accumulated micro
  }

  register(m) {
    const mech = m instanceof Mechanic ? m : new Mechanic(m);
    this.mechanics.set(mech.id, mech);
    return mech;
  }

  // Resolve all mechanics applicable to a scope with given scopeState.
  // ctxMeta: arbitrary read-only context (region cap, club mojo, etc.)
  // Returns { totalMicro:bigint, breakdown:[{id,kind,deltaMicro,rivalryTag}], rivalry:{tag:micro} }
  resolve(scope, scopeKey, scopeState = {}, ctxMeta = {}) {
    let total = 0n;
    const breakdown = [];
    // reset per-resolve rivalry accumulation (caller aggregates across scopes if desired)
    const localRivalry = new Map();

    for (const mech of this.mechanics.values()) {
      if (mech.scope !== scope && mech.scope !== 'world') continue;

      // once-per guard
      if (mech.oncePer) {
        const dedupe = mech.oncePer + ':' + scopeKey;
        let fired = this.firedOnce.get(mech.oncePer);
        if (!fired) { fired = new Set(); this.firedOnce.set(mech.oncePer, fired); }
        if (fired.has(dedupe)) continue;
        fired.add(dedupe);
      }

      const ctx = {
        seed: hashU64(mech.id + '|' + scopeKey).toString(),
        scopeState,
        meta: ctxMeta,
      };
      let delta = mech.compute(ctx); // bigint micro
      if (typeof delta !== 'bigint') delta = toMicro(delta);
      delta = delta * mech.weight; // integer linear stacking

      if (mech.stackable) {
        total = addMicro(total, delta);
      } else {
        // replace: keep the max absolute contribution
        total = (total > 0n ? total : 0n); // non-stacking simply adds once; design choice
        total = addMicro(total, delta);
      }

      breakdown.push({
        id: mech.id,
        kind: mech.kind,
        deltaMicro: delta.toString(),
        rivalryTag: mech.rivalryTag,
      });

      if (mech.rivalryTag) {
        const prev = localRivalry.get(mech.rivalryTag) || 0n;
        localRivalry.set(mech.rivalryTag, addMicro(prev, delta));
      }
    }

    const rivalry = {};
    for (const [tag, v] of localRivalry) rivalry[tag] = v.toString();
    return { totalMicro: total.toString(), breakdown, rivalry };
  }

  // Aggregate rivalry across many scopes into the engine's persistent matrix.
  accumulateRivalry(rivalryObj) {
    for (const [tag, v] of Object.entries(rivalryObj)) {
      const prev = this.rivalry.get(tag) || 0n;
      this.rivalry.set(tag, addMicro(prev, v));
    }
  }

  getRivalryMatrix() {
    const out = {};
    for (const [tag, v] of this.rivalry) out[tag] = v.toString();
    return out;
  }
}

// Convenience: derive a 0..1 float tint factor from an integer micro value for VIEW ONLY.
// Never used in ledger math.
export function microToTint(microStr, maxMicroStr) {
  const m = typeof microStr === 'bigint' ? microStr : BigInt(microStr || '0');
  const max = typeof maxMicroStr === 'bigint' ? maxMicroStr : BigInt(maxMicroStr || '1');
  if (max <= 0n) return 0;
  // integer ratio, then /1000 for a stable 0..1 view factor (still integer-div safe)
  return Number((m * 1000n) / max) / 1000;
}

export { MICRO };
export default { Mechanic, MechanicEngine, toMicro, addMicro, hashU64, microToTint, MICRO };
