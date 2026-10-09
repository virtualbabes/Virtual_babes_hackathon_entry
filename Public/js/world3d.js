// world3d.js — §25 Web-3D Client (Phase 1)
// Three.js scene that renders region capitals from /api/regions, applies the Mechanic
// framework for emergent vitality/rivalry tints, and auto-cycles the spectator feed via
// the EXISTING landing-page hook window.sendSpectate (game.js:334).
//
// No server changes: reuses GetRegionViews + sendSpectate. Registers enter3DWorld /
// enterMenuWorld consumed by Public/js/leaderboard_region.js (openLeaderboardRegion).
//
// CONSTITUTIONAL: ledger/score math is uint64 micro (mechanics.js). Three.js colors are
// VIEW-ONLY floats derived from integer micro via microToTint(). Never feed a float to logic.

import * as THREE from '/vendor/three.module.js';
import { MechanicEngine, microToTint, addMicro } from './mechanics.js';
import { DEFAULT_MECHANICS } from './mechanic_defs.js';

var API_BASE = '/api';

class World3DEngine {
  constructor() {
    this.mounted = false;
    this.engine = new MechanicEngine();
    DEFAULT_MECHANICS.forEach((m) => this.engine.register(m));
    this.regions = [];
    this.lastLobbyPlayers = [];
    this.spectatorIndex = 0;
    this.spectatorTimer = null;
    this.scene = null;
    this.camera = null;
    this.renderer = null;
    this.regionMeshes = new Map();
    this.overlayEl = null;
    this.containerEl = null;
    this.statusEl = null;
  }

  // ---- mount: called by window.enter3DWorld ----
  mount() {
    if (this.mounted) return;
    this.mounted = true;

    // overlay host (neon-glass, matches repo panel style)
    const html = `
<div id="world3d-overlay" class="overlay vbt-overlay" style="display:flex;flex-direction:column;background:rgba(10,10,20,0.92);">
  <div class="neon-glass-panel" style="flex:1;margin:12px;display:flex;flex-direction:column;overflow:hidden;">
    <div style="display:flex;justify-content:space-between;align-items:center;padding:10px 14px;border-bottom:1px solid rgba(255,255,255,0.08);">
      <h2 style="margin:0;color:#00f2fe;">🌐 World Explorer — §25</h2>
      <div>
        <button id="w3d-next" class="vbt-btn vbt-btn-secondary" style="font-size:11px;">⏭ Next Spectator</button>
        <button id="w3d-menu" class="vbt-btn vbt-btn-secondary" style="font-size:11px;" onclick="if(window.enterMenuWorld)window.enterMenuWorld()">🗂 Menu</button>
        <button id="w3d-close" class="vbt-btn vbt-btn-secondary" style="font-size:11px;">✕</button>
      </div>
    </div>
    <div id="w3d-canvas-host" style="flex:1;position:relative;min-height:300px;"></div>
    <div id="w3d-readout" class="neon-glass-panel" style="position:absolute;left:14px;bottom:14px;max-width:280px;padding:10px 12px;font-size:11px;color:#b0bec5;display:none;pointer-events:none;">
      <div id="w3d-readout-region" style="color:#00f2fe;font-weight:bold;margin-bottom:4px;"></div>
      <div id="w3d-readout-body"></div>
    </div>
    <div id="w3d-status" class="ai-status" style="padding:6px 14px;"></div>
  </div>
</div>`;
    document.body.insertAdjacentHTML('beforeend', html);
    this.overlayEl = document.getElementById('world3d-overlay');
    this.containerEl = document.getElementById('w3d-canvas-host');
    this.statusEl = document.getElementById('w3d-status');

    document.getElementById('w3d-next').addEventListener('click', () => this.cycleSpectator());
    document.getElementById('w3d-close').addEventListener('click', () => this.unmount());

    this._initThree();
    this._loop();
    this.refreshRegions();
    this._startSpectatorCycle();
    this.setStatus('World mounted. Reusing window.sendSpectate for fly-through.');
  }

  _initThree() {
    const host = this.containerEl;
    const w = host.clientWidth || 800;
    const h = host.clientHeight || 480;
    this.scene = new THREE.Scene();
    this.scene.background = new THREE.Color(0x0a0a14);
    this.camera = new THREE.PerspectiveCamera(60, w / h, 0.1, 1000);
    this.camera.position.set(0, 28, 48);
    this.camera.lookAt(0, 0, 0);
    this.renderer = new THREE.WebGLRenderer({ antialias: true });
    this.renderer.setSize(w, h);
    host.appendChild(this.renderer.domElement);

    // ambient + key light
    this.scene.add(new THREE.AmbientLight(0x4060ff, 0.6));
    const key = new THREE.PointLight(0x00f2fe, 1.2, 400);
    key.position.set(20, 40, 20);
    this.scene.add(key);

    // subtle starfield for depth
    const starGeo = new THREE.BufferGeometry();
    const starPos = new Float32Array(600 * 3);
    for (let i = 0; i < starPos?.length ?? 0; i++) starPos[i] = (Math.random() - 0.5) * 600;
    starGeo.setAttribute('position', new THREE.BufferAttribute(starPos, 3));
    this.scene.add(new THREE.Points(starGeo, new THREE.PointsMaterial({ color: 0x335577, size: 1.2 })));

    this._onResize = () => {
      const nw = host.clientWidth || 800, nh = host.clientHeight || 480;
      this.camera.aspect = nw / nh; this.camera.updateProjectionMatrix();
      this.renderer.setSize(nw, nh);
    };
    window.addEventListener('resize', this._onResize);

    // §25 Phase 2: click-to-warp raycaster on region meshes
    this.raycaster = new THREE.Raycaster();
    this.pointer = new THREE.Vector2();
    this._onCanvasClick = (ev) => {
      const rect = this.renderer.domElement.getBoundingClientRect();
      this.pointer.x = ((ev.clientX - rect.left) / rect.width) * 2 - 1;
      this.pointer.y = -((ev.clientY - rect.top) / rect.height) * 2 + 1;
      this.raycaster.setFromCamera(this.pointer, this.camera);
      const meshes = Array.from(this.regionMeshes.values()).map((e) => e.mesh);
      const hits = this.raycaster.intersectObjects(meshes, false);
      if (hits?.length ?? 0) {
        const mesh = hits[0].object;
        for (const [key, entry] of this.regionMeshes) {
          if (entry.mesh === mesh) { this._selectRegion(key); break; }
        }
      }
    };
    this.renderer.domElement.addEventListener('click', this._onCanvasClick);
  }

  // ---- fetch region views from existing server endpoint ----
  async refreshRegions() {
    try {
      const resp = await fetch(API_BASE + '/regions');
      const json = await resp.json();
      this.regions = (json && json.data) ? json.data : (Array.isArray(json) ? json : []);
      this._buildRegionMeshes();
    } catch (e) {
      this.setStatus('Region fetch failed: ' + e.message);
    }
  }

  _buildRegionMeshes() {
    // clear old
    for (const m of this.regionMeshes.values()) this.scene.remove(m.group);
    this.regionMeshes.clear();

    const n = Math.max(this.regions?.length ?? 0, 1);
    const radius = 30;
    this.regions.forEach((rv, i) => {
      const angle = (i / n) * Math.PI * 2;
      const x = Math.cos(angle) * radius;
      const z = Math.sin(angle) * radius;

      // resolve emergent vitality via mechanic framework (uint64 micro)
      const res = this.engine.resolve('region', rv.Region || ('r' + i), rv, {
        citizenCount: rv.Count || 0,
        cacheCount: (rv.Caches || [])?.length ?? 0,
        cap: rv.Cap || 1,
        mojoMicro: BigInt(rv.MojoMicro || 0),
        contestedMicro: BigInt(rv.ContestedMicro || 0),
        gravityMicro: BigInt(rv.GravityMicro || 0),
      });
      this.engine.accumulateRivalry(res.rivalry);
      const totalMicro = BigInt(res.totalMicro);

      // §30 Pet World: 3D-world power overlay — entity power caps/scales the region tower.
      // avg_entity_power is integer (0..600); derived from pet/bot stat vectors in the region.
      const ov = rv.EntityPowerOverlay;
      const powerH = ov ? Number(ov.avg_entity_power) : 0; // 0..600 integer
      const powerScale = 1 + Math.min(powerH, 600) / 300; // 1.0..3.0 (VIEW-ONLY float)

      // height = vitality (integer micro -> capped view height) scaled by power overlay
      const height = (4 + Number((totalMicro * 40n) / 100000000n)) * powerScale;
      const geo = new THREE.CylinderGeometry(3, 4, height, 16);
      // tint from rivalry (VIEW-ONLY float derived from integer micro) blended with power glow
      const rivalMicro = BigInt((res.rivalry.rivalry_aura || res.rivalry.region_vitality) || '0');
      const tint = microToTint(rivalMicro, 5000000n);
      const color = new THREE.Color().setHSL(0.55 - tint * 0.5, 0.7, 0.5);
      if (powerH > 0) color.setHSL(0.45 - tint * 0.4, 0.75, 0.4 + Math.min(powerH, 600) / 1200); // power glow
      const mat = new THREE.MeshStandardMaterial({ color, emissive: color.clone().multiplyScalar(0.25 + Math.min(powerH, 600) / 1200), metalness: 0.3, roughness: 0.6 });
      const mesh = new THREE.Mesh(geo, mat);
      mesh.position.set(x, height / 2, z);

      // label sprite
      const powTxt = ov ? (' · PWR ' + powerH + (ov.top_dominant_stat ? ' [' + ov.top_dominant_stat + ']' : '')) : '';
      const label = this._makeLabel((rv.Region || '#' + i) + ' · ' + (rv.Count || 0) + ' citizens' + powTxt);
      label.position.set(x, height + 2, z);

      const group = new THREE.Group();
      group.add(mesh); group.add(label);
      this.scene.add(group);
      this.regionMeshes.set(rv.Region || ('r' + i), { group, mesh, baseColor: color, totalMicro });
    });

    // gentle auto-rotate target
    this._targetCam = new THREE.Vector3(0, 24, 48);
  }

  _makeLabel(text) {
    const cv = document.createElement('canvas');
    cv.width = 256; cv.height = 64;
    const ctx = cv.getContext('2d');
    ctx.fillStyle = 'rgba(0,242,254,0.9)';
    ctx.font = '20px Rajdhani, sans-serif';
    ctx.fillText(text, 8, 40);
    const tex = new THREE.CanvasTexture(cv);
    const spr = new THREE.Sprite(new THREE.SpriteMaterial({ map: tex, transparent: true }));
    spr.scale.set(12, 3, 1);
    return spr;
  }

  // ---- spectator auto-cycle over existing hook ----
  setLobbyPlayers(players) {
    // called by network.js on lobby_update
    this.lastLobbyPlayers = Array.isArray(players) ? players : [];
  }

  _startSpectatorCycle() {
    if (this.spectatorTimer) clearInterval(this.spectatorTimer);
    this.spectatorTimer = setInterval(() => this.cycleSpectator(), 9000);
  }

  cycleSpectator() {
    const list = this.lastLobbyPlayers;
    if (!(list?.length ?? 0)) { this.setStatus('No active players to spectate yet.'); return; }
    // deterministic rotation (uint index, no Math.random in selection)
    this.spectatorIndex = (this.spectatorIndex + 1) % list?.length ?? 0;
    const p = list[this.spectatorIndex];
    const id = p && (p.id || p.wallet);
    if (!id) return;
    if (typeof window.sendSpectate === 'function') {
      window.sendSpectate(id); // EXISTING hook (game.js:334) — server streams match state
      this.setStatus('👁️ Spectating ' + String(id).slice(0, 10) + '  (' + (this.spectatorIndex + 1) + '/' + list?.length ?? 0 + ')');
      this._flyToRandomRegion();
    } else {
      this.setStatus('sendSpectate hook unavailable.');
    }
  }

  _flyToRandomRegion() {
    const keys = Array.from(this.regionMeshes.keys());
    if (!(keys?.length ?? 0)) return;
    const idx = this.spectatorIndex % keys?.length ?? 0;
    const entry = this.regionMeshes.get(keys[idx]);
    if (entry) {
      this._targetCam = entry.mesh.position.clone().add(new THREE.Vector3(0, 18, 26));
    }
  }

  // ---- §25 Phase 2: click-to-warp + live §27.7 term readout ----
  _selectRegion(key) {
    const entry = this.regionMeshes.get(key);
    if (!entry) return;
    // warp camera to the region capital
    this._targetCam = entry.mesh.position.clone().add(new THREE.Vector3(0, 16, 24));
    this.setStatus('Warped to region ' + key);
    this._fetchRegionDynamics(key);
  }

  async _fetchRegionDynamics(region) {
    try {
      const resp = await fetch(API_BASE + '/rivalry/world-dynamics?region=' + encodeURIComponent(region));
      const json = await resp.json();
      if (!json || !json.success) return;
      const box = document.getElementById('w3d-readout');
      const head = document.getElementById('w3d-readout-region');
      const body = document.getElementById('w3d-readout-body');
      if (!box || !head || !body) return;
      // Render the live §27.4 signature, highlighting the now-wired §27.7 terms.
      head.textContent = 'Region ' + (json.region || region) + '  ·  Score ' + json.score;
      body.innerHTML =
        'MarketVitality: ' + json.market_vitality + '<br>' +
        'CitizenGravity: ' + json.citizen_gravity + '<br>' +
        'ThemeCoherence: ' + json.theme_coherence + '<br>' +
        'EventDynamics: ' + json.event_dynamics + '<br>' +
        'ProfileImpact: ' + json.profile_impact + '<br>' +
        '<span style="color:#ffae42">DomesticCoherence (§27.7.1): ' + json.domestic_coherence + '</span><br>' +
        '<span style="color:#90caf9">EntityLegitimacy (§27.7.3): ' + json.entity_legitimacy + '</span><br>' +
        'FaithCoherence (§27.7): ' + json.faith_coherence + '<br>' +
        'RumorCoherence: ' + json.rumor_coherence + '<br>' +
        'EconomicPerk: ' + json.economic_perk;
      box.style.display = 'block';
    } catch (e) {
      this.setStatus('World-dynamics fetch failed: ' + e.message);
    }
  }

  _loop() {
    if (!this.mounted) return;
    const tick = () => {
      if (!this.mounted) return;
      // Skip the (expensive) Three.js render while the overlay is hidden so we never
      // burn GPU/CPU repainting an invisible scene.
      if (this.overlayEl && this.overlayEl.offsetParent === null) { requestAnimationFrame(tick); return; }
      // ease camera toward target
      if (this._targetCam) this.camera.position.lerp(this._targetCam, 0.02);
      this.camera.lookAt(0, 6, 0);
      // pulse region meshes by vitality (view-only)
      for (const { mesh, baseColor, totalMicro } of this.regionMeshes.values()) {
        const pulse = 1 + Math.sin(Date.now() / 600 + totalMicro) * 0.02;
        mesh.scale.y = pulse;
      }
      this.renderer.render(this.scene, this.camera);
      requestAnimationFrame(tick);
    };
    requestAnimationFrame(tick);
  }

  // ---- public: warp to a region capital by name (consumed by AI Citizens panel) ----
  warpToRegion(region) {
    if (!region || region === 'Base' || region === '') {
      this.setStatus('Citizen is in Base — no 3D capital to warp to.');
      return;
    }
    if (!this.mounted) this.mount();
    let matchKey = null;
    for (const key of this.regionMeshes.keys()) {
      if (String(key).toLowerCase() === String(region).toLowerCase()) { matchKey = key; break; }
    }
    if (!matchKey) {
      this.setStatus('Region "' + region + '" has no 3D capital yet (not populated).');
      return;
    }
    this._selectRegion(matchKey);
  }

  setStatus(m) { if (this.statusEl) this.statusEl.textContent = m || ''; }

  // ---- unmount: called by window.enterMenuWorld ----
  unmount() {
    if (!this.mounted) return;
    this.mounted = false;
    if (this.spectatorTimer) clearInterval(this.spectatorTimer);
    if (this._onResize) window.removeEventListener('resize', this._onResize);
    if (this.renderer) this.renderer.dispose();
    if (this.overlayEl && this.overlayEl.parentNode) this.overlayEl.parentNode.removeChild(this.overlayEl);
    this.regionMeshes.clear();
    this.setStatus('Returned to menu world.');
  }
}

// ---- singleton + global hooks (consumed by leaderboard_region.js) ----
const world = new World3DEngine();
window.World3DEngine = world;

window.enter3DWorld = function () { world.mount(); };
window.enterMenuWorld = function () { world.unmount(); };

// allow network.js to feed the spectator cycle the live lobby list
window.__setWorld3DLobbyPlayers = function (players) { world.setLobbyPlayers(players); };

export default world;
