// ============================================================================
// club_foundry.js — Club Creation & Management for World Dashboard
// ----------------------------------------------------------------------------
// Exposes club_service.go features: create club, manage members, treasury, staff
// Integrates with World Dashboard "Clubs" tab
// ============================================================================

var API_BASE = '/api';

// --- Club Types ---
const CLUB_TYPES = [
    { id: 'foundery', name: 'Foundery', icon: '🏭', desc: 'Manufacturing & production hub', cost: '5,000 $VBV' },
    { id: 'faith', name: 'Faith', icon: '⛪', desc: 'Religious organization & rituals', cost: '5,000 $VBV' },
    { id: 'trading', name: 'Trading', icon: '💰', desc: 'Commerce & market operations', cost: '5,000 $VBV' },
    { id: 'security', name: 'Security', icon: '🛡', desc: 'Defense & protection services', cost: '5,000 $VBV' },
];

// --- Staff Roles ---
const STAFF_ROLES = [
    { id: 'CEO', name: 'CEO', icon: '👑', desc: 'Full control' },
    { id: 'Manager', name: 'Manager', icon: '📋', desc: 'Day-to-day operations' },
    { id: 'Security', name: 'Security', icon: '🛡', desc: 'Defense & traps' },
    { id: 'Clerk', name: 'Clerk', icon: '📝', desc: 'Basic tasks' },
];

// --- State ---
let currentClub = null;
let myClubs = [];

// --- Init ---
export function initClubFoundry() {
    const el = document.getElementById('wd-clubs');
    if (!el) return;
    renderClubFoundry(el);
    fetchClubs(el);
}

async function fetchClubs(el) {
    try {
        const resp = await fetch(`${API_BASE}/clubs`);
        if (!resp.ok) throw new Error('Failed to fetch clubs');
        const data = await resp.json();
        myClubs = data.clubs || [];
        renderClubFoundry(el);
    } catch (e) {
        el.innerHTML = '<p style="color:#90a4ae;">Club data unavailable</p>';
    }
}

function renderClubFoundry(el) {
    let html = '';

    if (myClubs.length === 0) {
        // No clubs — show create form
        html = renderClubCreateForm();
    } else {
        // Has clubs — show management view
        html = renderClubManagement();
    }

    el.innerHTML = html;
    attachEventListeners(el);
}

function renderClubCreateForm() {
    return `
        <div class="club-foundry-container">
            <div class="club-foundry-header">
                <h3 style="color:#e5e7eb;margin:0 0 8px;">🏛️ Found a Club</h3>
                <p style="color:#90a4ae;font-size:11px;margin:0 0 16px;">
                    Create your own organization. Clubs can own territories, hire staff, and participate in the economy.
                </p>
            </div>

            <!-- Club Type Selection -->
            <div class="club-types-grid">
                ${CLUB_TYPES.map(ct => `
                    <div class="club-type-card" data-type="${ct.id}">
                        <span class="club-type-icon">${ct.icon}</span>
                        <span class="club-type-name">${ct.name}</span>
                        <span class="club-type-desc">${ct.desc}</span>
                        <span class="club-type-cost">${ct.cost}</span>
                    </div>
                `).join('')}
            </div>

            <!-- Create Form -->
            <div class="club-create-form">
                <div class="club-form-row">
                    <label>Club Name</label>
                    <input type="text" id="club-name-input" placeholder="Enter club name..." maxlength="32" />
                </div>
                <div class="club-form-row">
                    <label>Territory ID (optional)</label>
                    <input type="text" id="club-territory-input" placeholder="e.g., region-1" />
                </div>
                <div class="club-form-row">
                    <label>Network</label>
                    <select id="club-network-select">
                        <option value="Voi">Voi Mainnet</option>
                        <option value="Algorand">Algorand</option>
                    </select>
                </div>
                <div class="club-form-actions">
                    <button class="vbt-btn vbt-btn-primary" id="club-create-btn">
                        🏛️ Found Club (5,000 $VBV)
                    </button>
                </div>
            </div>

            <!-- Info -->
            <div class="club-foundry-info">
                <h4 style="color:#e5e7eb;">Club Benefits</h4>
                <ul>
                    <li>🏛️ Own territories and collect taxes</li>
                    <li>👥 Hire staff (Manager, Security, Clerk)</li>
                    <li>🛡️ Deploy traps and defenses</li>
                    <li>💰 Earn from club activities</li>
                    <li>⚔️ Enter tournaments as a club</li>
                </ul>
            </div>
        </div>
    `;
}

function renderClubManagement() {
    return `
        <div class="club-management-container">
            ${myClubs.map(club => `
                <div class="club-card" data-club-id="${club.ID || club.id}">
                    <div class="club-card-header">
                        <div class="club-card-info">
                            <span class="club-card-name">${escapeHtml(club.Name || club.name)}</span>
                            <span class="club-card-type">${escapeHtml(club.Type || club.type || 'Standard')}</span>
                        </div>
                        <span class="club-card-mojo">⚡ ${club.Mojo || club.mojo || 0}</span>
                    </div>

                    <!-- Stats -->
                    <div class="club-stats-row">
                        <div class="club-stat-mini">
                            <span class="club-stat-label">Members</span>
                            <span class="club-stat-val">${Object.keys(club.Members || club.members || {}).length}</span>
                        </div>
                        <div class="club-stat-mini">
                            <span class="club-stat-label">Treasury</span>
                            <span class="club-stat-val">${formatVBV(club.Treasury || club.treasury || 0)}</span>
                        </div>
                        <div class="club-stat-mini">
                            <span class="club-stat-label">Territories</span>
                            <span class="club-stat-val">${(club.Territories || club.territories || []).length}</span>
                        </div>
                        <div class="club-stat-mini">
                            <span class="club-stat-label">Staff</span>
                            <span class="club-stat-val">${Object.keys(club.Staff || club.staff || {}).length}</span>
                        </div>
                    </div>

                    <!-- Staff -->
                    <div class="club-staff-section">
                        <h5>Staff</h5>
                        <div class="club-staff-list">
                            ${Object.entries(club.Staff || club.staff || {}).map(([wallet, role]) => `
                                <div class="club-staff-item">
                                    <span class="staff-role-badge role-${role.toLowerCase()}">${role}</span>
                                    <span class="staff-wallet">${wallet.slice(0, 6)}...${wallet.slice(-4)}</span>
                                </div>
                            `).join('')}
                        </div>
                    </div>

                    <!-- Members -->
                    <div class="club-members-section">
                        <h5>Members</h5>
                        <div class="club-members-list">
                            ${Object.keys(club.Members || club.members || {}).map(wallet => `
                                <span class="club-member-badge">${wallet.slice(0, 6)}...${wallet.slice(-4)}</span>
                            `).join('')}
                        </div>
                    </div>

                    <!-- Actions -->
                    <div class="club-actions-row">
                        <button class="vbt-btn vbt-btn-primary" onclick="window.clubAddMember('${club.ID || club.id}')">
                            👥 Add Member
                        </button>
                        <button class="vbt-btn vbt-btn-secondary" onclick="window.clubManageStaff('${club.ID || club.id}')">
                            👑 Manage Staff
                        </button>
                        <button class="vbt-btn vbt-btn-secondary" onclick="window.clubViewInventory('${club.ID || club.id}')">
                            📦 Inventory
                        </button>
                    </div>
                </div>
            `).join('')}
        </div>
    `;
}

function attachEventListeners(el) {
    // Club type selection
    el.querySelectorAll('.club-type-card').forEach(card => {
        card.addEventListener('click', () => {
            el.querySelectorAll('.club-type-card').forEach(c => c.classList.remove('selected'));
            card.classList.add('selected');
        });
    });

    // Create button
    const createBtn = el.querySelector('#club-create-btn');
    if (createBtn) {
        createBtn.addEventListener('click', () => {
            const name = el.querySelector('#club-name-input')?.value.trim();
            const territory = el.querySelector('#club-territory-input')?.value.trim();
            const network = el.querySelector('#club-network-select')?.value || 'Voi';
            if (!name) {
                alert('Please enter a club name');
                return;
            }
            createClub(name, territory, network);
        });
    }
}

async function createClub(name, territory, network) {
    const payload = {
        name: name,
        type: 'standard',
        territory_id: territory || '',
        network: network,
        txid: ''
    };
    try {
        const resp = await fetch(`${API_BASE}/clubs/create`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });
        if (!resp.ok) throw new Error('Failed to create club');
        const data = await resp.json();
        if (window.showToast) window.showToast(`🏛️ Club '${name}' founded!`, 'success');
        initClubFoundry();
    } catch (e) {
        if (window.showToast) window.showToast(`❌ ${e.message}`, 'error');
    }
}

// --- Utility ---
function escapeHtml(s) {
    return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({
        '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
    }[c]));
}

function formatVBV(micro) {
    const vbv = Number(micro) / 1_000_000;
    if (vbv >= 1_000_000) return `${(vbv / 1_000_000).toFixed(1)}M`;
    if (vbv >= 1_000) return `${(vbv / 1_000).toFixed(1)}K`;
    return vbv.toFixed(2);
}

// --- Global handlers ---
window.clubAddMember = function(clubId) {
    const wallet = prompt('Enter member wallet address:');
    if (!wallet) return;
    // TODO: Call API to add member
    alert(`Adding ${wallet} to club ${clubId}`);
};

window.clubManageStaff = function(clubId) {
    alert(`Managing staff for club ${clubId}`);
};

window.clubViewInventory = function(clubId) {
    alert(`Viewing inventory for club ${clubId}`);
};

window.initClubFoundry = initClubFoundry;
