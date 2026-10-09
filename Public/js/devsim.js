// devsim.js — DEV-ONLY state simulation (no chain, no network).
// Loaded ONLY when the URL contains ?devsim=1 (see index.html bootstrap).
// Purpose: let the UI render real content locally since live game state is
// supplied externally (gzip-note queries) and is absent on the bare dev server.
// Reversible: remove ?devsim=1 and the app uses the real WASM bridge + live state.
// No on-chain calls, no $VBV spent.

(function () {
  'use strict';
  if (!location.search.includes('devsim')) return;

  const DEV_WALLET = 'DEVWALLET0000000000000000000000000000000000000000XYSIM';

  // --- Shop registry (mirrors economy.js GlobalShopRegistry shape) ----------
  const ShopRegistry = {
    elemental_core:   { name: 'Elemental Core',   desc: 'Fire/Ice/Volt tuning matrix', price: 120, requiredMojo: 0 },
    tactical_chip:    { name: 'Tactical Chip',    desc: 'Combat logic overclock',       price: 200, requiredMojo: 50 },
    vitality_serum:   { name: 'Vitality Serum',   desc: 'Stamina reconstruction',        price: 90,  requiredMojo: 0 },
    hardware_rig:     { name: 'Hardware Rig',     desc: 'Mining/compute module',         price: 350, requiredRole: 'Engineer' },
    stamina_stim:     { name: 'Stamina Stim',     desc: 'Fatigue reducer',               price: 60,  requiredMojo: 0 },
  };

  // --- Clubs with inventory per shop category -------------------------------
  const DevClubs = {
    club_elem:  { id: 'club_elem',  name: 'Pyre Syndicate',  type: 'Elemental', owner: DEV_WALLET, territories: ['the_lab'], mojo: 120, inventory: { elemental_core: 8, stamina_stim: 12 } },
    club_tac:   { id: 'club_tac',   name: 'Wardens',        type: 'Tactical',  owner: DEV_WALLET, territories: ['north_district'], mojo: 80, inventory: { tactical_chip: 5, stamina_stim: 6 } },
    club_vit:   { id: 'club_vit',   name: 'Lifewell',       type: 'Vitality',  owner: DEV_WALLET, territories: ['south_slums'], mojo: 40, inventory: { vitality_serum: 9 } },
    club_hw:    { id: 'club_hw',    name: 'Forgeworks',     type: 'Hardware',  owner: DEV_WALLET, territories: ['west_port','data_haven'], mojo: 200, inventory: { hardware_rig: 3, tactical_chip: 2 } },
  };

  // --- Player/game state (mirrors GetGameState() shape the UI reads) --------
  const DevState = {
    job_role: 'Engineer',
    level: 7,
    reputation: 1840,
    employer_id: 'club_hw',
    clubs: DevClubs,
    pets: [{ id: 'pet_1', name: 'Mochi' }, { id: 'pet_2', name: 'Bolt' }],
    life_assets: [{ id: 'life_1', name: 'Companion A' }],
    ai_citizens: [{ id: 'cit_1', name: 'Aria' }, { id: 'cit_2', name: 'Kano' }],
    citizens: [{ id: 'cit_1', name: 'Aria' }, { id: 'cit_2', name: 'Kano' }],
    world_events: [{ id: 'ev_1', name: 'Neon Storm' }, { id: 'ev_2', name: 'Black Market Riot' }],
    active_events: [{ id: 'ev_1', name: 'Neon Storm' }],
    spectatorMatchState: null,
    replay_state: 'SYNCHRONIZED',
  };

  // Expose for the bridge-shaped consumers (hub, etc.)
  window.GetGameState = function (scope) {
    if (scope && DevState[scope] !== undefined) return DevState[scope];
    return DevState;
  };

  // Expose for economy.js shop rendering (read via window.__dev* fallback)
  window.__devClubs = DevClubs;
  window.__devShopRegistry = ShopRegistry;
  window.userAddress = DEV_WALLET;

  // Seed a couple of lobby players so the hub Lobby panel isn't empty
  window.lastLobbyPlayers = [
    { id: 'player_1', is_admin: false },
    { id: 'player_2', is_admin: true },
  ];
  window.myClientId = 'dev_self';

  console.log('[DEV-SIM] state seeded (no chain). Remove ?devsim=1 for live state.');
})();
