// ============================================================================
// career_tree.js — CAREER PATH / CIVIL RANK / PROMOTION / DEMOTION
// ----------------------------------------------------------------------------
// THE RULE THIS MODULE FOLLOWS: it RENDERS the served taxonomy and re-declares
// none of it.
//
// It used to DECLARE one in the client — a 12-pathway list carrying
// `faction: 'JUSTICE' | 'UNDERWORLD' | 'HYBRID'` plus a tier table copied from
// the Go constants — and the server had never served a single one of those
// strings, so the panel showed a system the engine did not implement. That is the
// fabricated-taxonomy class this repository has had to repair more than once.
//
// Everything below comes from GET /api/career/path. The only two things this
// client may NAME are the path it chooses and the role it promotes into. The
// civil rank, the eligibility, whether a promotion is earned, whether a career is
// demoted and every requirement are DERIVED by `career_path.go`.
//
//   GET  /api/career/path          the whole system for the connected wallet
//   POST /api/career/path/choose   {path}  the one choice, made once
//   POST /api/career/promote       {role}  the level-cap promotion
// ============================================================================

var API_BASE = '/api';

let served = null;   // the last successful GET /api/career/path payload
let busy = false;    // one in-flight action at a time

// api(): the repository convention — the helper prepends API_BASE, so a caller
// passes a path WITHOUT the /api prefix. The startsWith guard keeps a
// fully-prefixed path working too, which is the pattern world_dashboard uses.
function api(path, opts) {
    const url = path.startsWith('/api/') ? path : API_BASE + path;
    return fetch(url, opts);
}

function walletForRequest() {
    if (typeof window.getWalletAddress === 'function') {
        const w = window.getWalletAddress();
        if (w) return w;
    }
    return window.currentWallet || window.userAddress || '';
}

function esc(s) {
    return String(s == null ? '' : s).replace(/[&<>"']/g, function (c) {
        return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c];
    });
}

// Integer-only money formatting: the VALUE is micro-units and stays an integer;
// only its presentation divides.
function fmtVBV(micro) {
    const n = Number(micro || 0);
    if (!isFinite(n) || n <= 0) return '0 $VBV';
    if (n >= 1e12) return (n / 1e12).toFixed(1) + 'M $VBV';
    if (n >= 1e9) return (n / 1e9).toFixed(1) + 'K $VBV';
    return (n / 1e6).toFixed(0) + ' $VBV';
}

// PRESENTATION ONLY. The server owns which paths exist; this owns what colour
// each is drawn in, and an unknown id simply gets the neutral tint.
function pathColour(id) {
    if (id === 'justice') return '#2196f3';
    if (id === 'criminal') return '#f44336';
    return '#ff9800';
}

function relationColour(rel) {
    if (rel === 'direct') return '#f44336';
    if (rel === 'interpreted') return '#ff9800';
    if (rel === 'shared') return '#4caf50';
    return '#607d8b';
}

// ── Entry points ─────────────────────────────────────────────────────────────

// initCareerTree: the World Dashboard "Career" tab EMBEDS this into #wd-career.
export function initCareerTree() {
    const el = document.getElementById('wd-career');
    if (!el) return;
    loadCareerPath(el);
}

// openCareers: the standalone overlay, reached from the World Dashboard via
// WD_ROUTES.career. It owns its own root and follows the contained-overlay
// contract (revealed with INLINE display, so it must never receive `.hidden`).
export function openCareers() {
    if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
    let overlay = document.getElementById('careers-overlay');
    if (!overlay) {
        document.body.insertAdjacentHTML('beforeend', `
<div id="careers-overlay" class="overlay vbt-overlay" style="display:none;">
    <div class="neon-glass-panel careers-panel">
        <button id="btn-close-careers" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
        <h2>Career Path &amp; Progression</h2>
        <div id="careers-content"></div>
    </div>
</div>`);
        overlay = document.getElementById('careers-overlay');
        document.getElementById('btn-close-careers').addEventListener('click', function () {
            overlay.style.display = 'none';
        });
    }
    overlay.style.display = 'flex';
    loadCareerPath(document.getElementById('careers-content'));
}
window.openCareers = openCareers;

// ── The served read ──────────────────────────────────────────────────────────

async function loadCareerPath(el) {
    if (!el) return;
    const wallet = walletForRequest();
    if (!wallet) {
        // A refused read is STATED, never rendered as "you have nothing".
        el.innerHTML = '<div class="career-note">Connect your wallet to read your career path, ' +
            'civil rank and promotion gates.</div>';
        return;
    }
    el.innerHTML = '<div class="career-note">Reading the career system...</div>';
    try {
        const resp = await api('/career/path?wallet=' + encodeURIComponent(wallet));
        if (resp.status === 429) {
            // Rate-limited: retry ONCE, then SAY so rather than showing an empty panel.
            await new Promise(function (r) { setTimeout(r, 450); });
            const retry = await api('/career/path?wallet=' + encodeURIComponent(wallet));
            if (!retry.ok) { renderRefused(el, 'rate-limited'); return; }
            const r2 = await retry.json();
            served = r2.data || null;
            render(el);
            return;
        }
        if (!resp.ok) { renderRefused(el, 'HTTP ' + resp.status); return; }
        const payload = await resp.json();
        served = payload.data || null;
        render(el);
    } catch (e) {
        renderRefused(el, String(e && e.message ? e.message : e));
    }
}

function renderRefused(el, why) {
    el.innerHTML = '<div class="career-note" style="color:#ffb74d;">The career system could not be read (' +
        esc(why) + '). This is a READ failure, not an empty career.</div>';
}

// ── Render ───────────────────────────────────────────────────────────────────

function render(el) {
    if (!served) { renderRefused(el, 'no data'); return; }
    const s = served;
    let html = '<div class="career-tree-container">';

    // 1. WHERE THE PLAYER STANDS: the path held, the civil rank, and any warning.
    html += '<div class="career-status-panel">';
    if (s.path) {
        html += '<div class="career-stat"><span class="career-stat-label">Path</span>' +
            '<span class="career-stat-val" style="color:' + pathColour(s.path) + '">' + esc(s.path) + '</span></div>';
    } else {
        html += '<div class="career-stat"><span class="career-stat-label">Path</span>' +
            '<span class="career-stat-val" style="color:#90a4ae">not chosen</span></div>';
    }
    html += '<div class="career-stat"><span class="career-stat-label">Civil rank</span>' +
        '<span class="career-stat-val">' + esc(s.civil ? s.civil.tier : 'user') + '</span></div>' +
        '<div class="career-stat"><span class="career-stat-label">Shops / territories</span>' +
        '<span class="career-stat-val">' + esc((s.civil ? s.civil.shops : 0)) + ' / ' + esc((s.civil ? s.civil.territories : 0)) + '</span></div>';
    if (s.civil && s.civil.region_opened) {
        html += '<div class="career-stat"><span class="career-stat-label">Region</span>' +
            '<span class="career-stat-val" style="color:#4caf50">' + esc(s.civil.region) + ' (opened)</span></div>';
    }
    html += '<div class="career-stat"><span class="career-stat-label">Promoted careers</span>' +
        '<span class="career-stat-val">' + esc((s.promoted_roles || []).length) + '</span></div>';
    html += '</div>';

    // An outstanding warning is the FIRST thing a player must see, with the amount to earn back.
    if (s.warning) {
        const w = s.warning;
        html += '<div class="career-warning">' +
            '<strong>Career at risk:</strong> ' + esc(w.role) + ' needs ' + fmtVBV(w.required_micro) +
            ' sustained; you hold ' + fmtVBV(w.avg_micro) + ' (' + fmtVBV(w.shortfall_micro) + ' short).' +
            (w.grace_ends_at ? ' Earn it back before ' + esc(new Date(w.grace_ends_at).toLocaleString()) +
                ' or the career is demoted.' : '') +
            '</div>';
    }
    if ((s.demotions || []).length) {
        html += '<h3>Demotion history</h3><div class="career-demotions">';
        s.demotions.forEach(function (d) {
            html += '<div class="career-demotion">' + esc(d.role) + ' — required ' + fmtVBV(d.required_micro) +
                ', held ' + fmtVBV(d.avg_micro) + ' (' + fmtVBV(d.shortfall_micro) + ' short). ' +
                '<span style="color:#90a4ae">' + esc(d.reason) + '</span></div>';
        });
        html += '</div>';
    }

    // 2. THE CIVIL LADDER, with what unlocks each rung (served, not restated).
    html += '<h3>Civil rank</h3><div class="career-civil-ladder">';
    (s.civil_ranks || []).forEach(function (r) {
        const active = s.civil && s.civil.tier === r.tier;
        html += '<div class="career-civil-rank' + (active ? ' is-active' : '') + '">' +
            '<span class="career-civil-tier">' + esc(r.tier) + '</span>' +
            '<span class="career-civil-req">' + esc(r.requires) + '</span></div>';
    });
    html += '</div>';

    html += renderPathChoice(s);
    html += renderPathMatrix(s);
    html += renderCareers(s);
    html += renderUpgrades(s);
    html += renderMatrices(s);
    html += renderRules(s);
    html += '</div>';

    el.innerHTML = html;

    // ONE delegated listener, assigned (not added) so a re-render replaces it
    // instead of stacking a second handler on every refresh.
    el.onclick = function (ev) {
        const t = ev.target;
        if (!t || !t.dataset) return;
        if (t.dataset.careerPath) { window.chooseCareerPath(t.dataset.careerPath); return; }
        if (t.dataset.careerRole) { window.promoteCareerRole(t.dataset.careerRole); return; }
        if (t.dataset.careerStaffWallet) {
            window.requestStaffCareerUpgrade(t.dataset.careerStaffWallet, t.dataset.careerStaffRole);
        }
    };
}

// 3. THE CHOICE. Offered ONLY when the server says it is eligible, and the
// refusal reason is the SERVER's own sentence (territories held, region state).
function renderPathChoice(s) {
    const e = s.eligibility || {};
    let html = '<h3>Choose your career path</h3>';
    if (s.path) {
        html += '<div class="career-note">Your path is <strong style="color:' + pathColour(s.path) + '">' +
            esc(s.path) + '</strong>. A path is chosen once; it is not a switch.</div>';
        return html;
    }
    if (!e.eligible) {
        html += '<div class="career-note" style="color:#ffb74d;">Not yet available: ' + esc(e.reason || 'locked') +
            ' (territories held: ' + esc(e.territories == null ? 0 : e.territories) +
            ', region opened: ' + (e.region_opened ? 'yes' : 'no') + ').</div>';
        return html;
    }
    html += '<div class="career-path-grid">';
    (s.paths || []).forEach(function (p) {
        html += '<div class="career-path-card" style="border-color:' + pathColour(p.id) + '">' +
            '<div class="career-path-name" style="color:' + pathColour(p.id) + '">' + esc(p.label) + '</div>' +
            '<div class="career-path-summary">' + esc(p.summary) + '</div>' +
            // A data- attribute plus ONE delegated listener, never a concatenated
            // onclick string: the value is server-authored and must not be spliced
            // into JS source, and the handler gate resolves names in markup.
            '<button class="vbt-btn career-pick-path" data-career-path="' + esc(p.id) + '">Take the ' +
            esc(p.label) + ' path</button>' +
            '</div>';
    });
    html += '</div>';
    return html;
}

// 4. THE PATH MATRIX, cell by cell, from what the server resolved — the client
// never computes a relation.
function renderPathMatrix(s) {
    let html = '<h3>Path rivalry</h3><div class="career-path-matrix">';
    (s.paths || []).forEach(function (p) {
        html += '<div class="career-matrix-row"><span class="career-matrix-head" style="color:' +
            pathColour(p.id) + '">' + esc(p.label) + '</span>';
        (p.rivals || []).forEach(function (r) {
            html += '<span class="career-matrix-cell" title="' + esc(r.explain) + '">' +
                '<span style="color:' + relationColour(r.relation) + '">' + esc(r.relation) + '</span>' +
                ' vs ' + esc(r.against) + (r.bonus_bps ? ' (+' + (r.bonus_bps / 100) + '%)' : '') +
                '</span>';
        });
        html += '</div>';
    });
    html += '</div>';
    return html;
}

// 5. THE CAREER TABLE. Every declared career, its gate, and — for a locked one —
// the ONE missing piece, quoted from the server.
function renderCareers(s) {
    let html = '<h3>Careers</h3><div class="career-table">';
    (s.careers || []).forEach(function (c) {
        const held = c.promoted;
        html += '<div class="career-row' + (held ? ' is-held' : '') + '">' +
            '<span class="career-role">' + esc(c.role) + '</span>' +
            '<span class="career-home" style="color:' + pathColour(c.home_path) + '">' + esc(c.home_path) + '</span>' +
            '<span class="career-gate">level ' + esc(c.min_lesson_level) +
            ' · tier ' + esc(c.min_role_tier) +
            ' · ' + fmtVBV(c.min_sustained_micro) +
            ' · ' + esc(c.min_civil_tier) + '</span>' +
            '<span class="career-progress">you: level ' + esc(c.lesson_level) +
            ' · tier ' + esc(c.role_tier) + ' · ' + fmtVBV(c.sustained_micro) + '</span>';
        if (held) {
            html += '<span class="career-held-badge">HELD</span>';
        } else if (c.eligible) {
            // UNLOCKED, NOT MANDATORY. The server reports that every gate for this career is met;
            // the upgrade is still the PLAYER's to take, so the control is an offer and the reason
            // is the SERVER's own sentence (never a client-invented "you must").
            // data- attribute + the delegated listener (see renderPathChoice).
            const unlockHint = (s.upgrades && s.upgrades.statement) || 'Unlocked to upgrade.';
            html += '<span class="career-unlocked-badge" title="' + esc(unlockHint) + '">UNLOCKED</span>' +
                '<button class="vbt-btn career-pick-role" data-career-role="' + esc(c.role) + '">Upgrade</button>';
        } else {
            html += '<span class="career-missing">' + esc(c.missing || 'locked') + '</span>';
        }
        html += '</div>';
    });
    html += '</div>';
    return html;
}

// 5b. THE ADVISORY — "unlocked, never forced".
//
// The server states what is UNLOCKED to upgrade and who the caller may ask; this renders exactly
// that. Three properties are visible in the markup on purpose:
//   - the statement and the basis are QUOTED, never restated (so the panel cannot claim an upgrade
//     is mandatory when the server says it is optional);
//   - an unlock with no staff section says so HONESTLY ("nothing is unlocked yet" / "you employ
//     nobody") instead of rendering an empty box;
//   - the staff control says REQUEST, because an employer may ask and only the staff member may act.
function renderUpgrades(s) {
    const u = s.upgrades;
    if (!u) return '';
    let html = '<h3>Upgrades — unlocked, not forced</h3><div class="career-upgrade-advisory">';
    html += '<div class="career-unlock-statement">' + esc(u.statement || '') + '</div>' +
        '<div class="career-unlock-basis">Unlock: ' + esc(u.unlock_basis || '') + '</div>' +
        '<div class="career-unlock-basis">Notice: ' + esc(u.notice_rule || '') + '</div>' +
        '<div class="career-unlock-basis">This upgrade is optional: ' +
        (u.is_optional === true ? 'yes — take it or leave it' : 'no') + '</div>';

    const unlocked = u.unlocked_roles || [];
    if (unlocked.length) {
        html += '<div class="career-unlock-list"><span class="career-unlock-list-label">Unlocked to upgrade now</span>' +
            unlocked.map(function (r) { return '<span class="career-unlock-role">' + esc(r) + '</span>'; }).join('') +
            '</div>';
    } else {
        html += '<div class="career-unlock-none">Nothing is unlocked to upgrade yet — the career table ' +
            'above states the one gate each career is waiting on.</div>';
    }

    // REQUESTS ADDRESSED TO THE CALLER. A request names one wallet, so this is the caller's own
    // employer's request and never another player's.
    (u.requests_to_me || []).forEach(function (q) {
        html += '<div class="career-upgrade-request-note">' + esc(q.message || '') +
            ' <span class="career-upgrade-request-meta">' + esc(q.club_name || '') + ' · ' + esc(q.role || '') +
            '</span></div>';
    });

    // THE STAFF PROJECTION. The BASIS is always shown, so a reader knows where "staff" came from.
    html += '<div class="career-staff-basis">' + esc(u.staff_basis || '') + '</div>';
    const staff = u.staff || [];
    if (staff.length) {
        html += '<h4>Your staff — ' + esc(u.staff_ready_count == null ? 0 : u.staff_ready_count) + ' of ' +
            esc(staff.length) + ' ready to upgrade</h4><div class="career-staff">';
        staff.forEach(function (st) {
            html += '<div class="career-staff-row">' +
                '<span class="career-staff-wallet">' + esc(st.wallet) + '</span>' +
                '<span class="career-staff-role">' + esc(st.club_role || '') +
                (st.job_role ? ' · ' + esc(st.job_role) : '') + '</span>' +
                '<span class="career-staff-level">level ' + esc(st.lesson_level) + '</span>';
            if (st.upgrade_ready) {
                html += '<span class="career-staff-ready">UNLOCKED: ' +
                    (st.unlocked_roles || []).map(function (r) { return esc(r); }).join(', ') + '</span>';
            } else {
                html += '<span class="career-staff-waiting">no upgrade unlocked yet</span>';
            }
            // One control per unlocked role. A repeat is harmless: the server records ONE request per
            // (staff, role) and REPORTS a repeat instead of re-notifying, so a nudge cannot become
            // pressure. `requested_role` is SERVED state, never a client guess.
            (st.unlocked_roles || []).forEach(function (role) {
                html += '<button class="vbt-btn career-staff-request" data-career-staff-wallet="' + esc(st.wallet) +
                    '" data-career-staff-role="' + esc(role) + '">Request ' + esc(role) + '</button>';
            });
            if (st.requested_role) {
                html += '<span class="career-staff-requested">already requested: ' + esc(st.requested_role) +
                    (st.requested_at ? ' (' + esc(new Date(st.requested_at).toLocaleString()) + ')' : '') + '</span>';
            }
            html += '</div>';
        });
        html += '</div>';
    } else {
        html += '<div class="career-unlock-none">You employ nobody, so there is no staff to request an ' +
            'upgrade for. Staff are the wallets on the roster of a club you own.</div>';
    }
    html += '</div>';
    return html;
}

// 6. THE THREE MATRICES, each with its OWNER, so a number is never shown without
// the layer it came from.
function renderMatrices(s) {
    let html = '<h3>Rival matrices</h3><div class="career-matrices">';
    (s.matrices || []).forEach(function (m) {
        html += '<div class="career-matrix"><span class="career-matrix-stage">' + esc(m.stage) + '</span>' +
            '<span class="career-matrix-owner">' + esc(m.owner) + '</span>' +
            '<span class="career-matrix-desc">' + esc(m.describes) + '</span></div>';
    });
    html += '</div>';
    if ((s.career_pairs || []).length) {
        html += '<h4>Career pair deltas</h4><div class="career-pairs">';
        s.career_pairs.forEach(function (p) {
            html += '<span class="career-pair">' + esc(p.pair) + ' <span style="color:' +
                (p.delta < 0 ? '#f44336' : '#4caf50') + '">' + esc(p.delta) + '</span></span>';
        });
        html += '</div>';
    }
    return html;
}

// 7. THE RULES, quoted from the server rather than restated here.
function renderRules(s) {
    if (!s.rules) return '';
    let html = '<h3>Rules (served)</h3><div class="career-rules">';
    Object.keys(s.rules).forEach(function (k) {
        const v = s.rules[k];
        html += '<div class="career-rule"><span class="career-rule-key">' + esc(k) + '</span>' +
            '<span class="career-rule-val">' + esc(v) + '</span></div>';
    });
    html += '</div>';
    return html;
}

// ── The two actions ──────────────────────────────────────────────────────────
//
// These are the ONLY two things this client may name. Both send a single field
// (`path` / `role`); the server decodes with DisallowUnknownFields, so a body
// carrying a civil rank, a promotion list or a balance cannot even be parsed.
// Every outcome — including a refusal — is reported as the SERVER's own words.

function tell(msg, kind) {
    if (typeof window.showToast === 'function') window.showToast(msg, kind || 'info');
}

async function postJSON(path, payload, wallet) {
    return api(path + '?wallet=' + encodeURIComponent(wallet), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
    });
}

async function refresh() {
    const el = document.getElementById('careers-content') || document.getElementById('wd-career');
    await loadCareerPath(el);
}

async function chooseCareerPath(path) {
    if (busy) return;
    const wallet = walletForRequest();
    if (!wallet) { tell('Connect your wallet first.', 'error'); return; }
    busy = true;
    try {
        const resp = await postJSON('/career/path/choose', { path: path }, wallet);
        const body = await resp.json().catch(function () { return {}; });
        if (!resp.ok || body.success === false) {
            tell('Path not chosen: ' + (body.error || ('HTTP ' + resp.status)), 'error');
        } else {
            tell('Career path chosen: ' + (body.path || path), 'success');
        }
    } catch (e) {
        tell('Path not chosen: ' + String(e && e.message ? e.message : e), 'error');
    } finally {
        busy = false;
        await refresh();
    }
}

async function promoteCareerRole(role) {
    if (busy) return;
    const wallet = walletForRequest();
    if (!wallet) { tell('Connect your wallet first.', 'error'); return; }
    busy = true;
    try {
        const resp = await postJSON('/career/promote', { role: role }, wallet);
        const body = await resp.json().catch(function () { return {}; });
        if (!resp.ok || body.success === false) {
            // The refusal carries the requirements object, and its `missing` is the
            // one unmet gate stated with its exact shortfall.
            const missing = (body.requirements && body.requirements.missing) || body.error || ('HTTP ' + resp.status);
            tell('Not promoted: ' + missing, 'error');
        } else {
            tell('Promoted: ' + (body.role || role), 'success');
        }
    } catch (e) {
        tell('Not promoted: ' + String(e && e.message ? e.message : e), 'error');
    } finally {
        busy = false;
        await refresh();
    }
}

// THE EMPLOYER'S ACTION. It ASKS, it does not grant: the server refuses anything the staff member
// is not already UNLOCKED for, records ONE request against the employer's own club and notifies the
// employee. The employee's career is untouched, so this control can never promote anybody.
async function requestStaffCareerUpgrade(staffWallet, role) {
    if (busy) return;
    const wallet = walletForRequest();
    if (!wallet) { tell('Connect your wallet first.', 'error'); return; }
    busy = true;
    try {
        const resp = await postJSON('/career/staff/request', { staff_wallet: staffWallet, role: role }, wallet);
        const body = await resp.json().catch(function () { return {}; });
        if (!resp.ok || body.success === false) {
            // The refusal is the SERVER's sentence: "not your staff", "not unlocked yet: <the one
            // missing gate>", or "already holds". Never re-worded here.
            tell('Request not sent: ' + (body.error || ('HTTP ' + resp.status)), 'error');
        } else {
            tell((body.request && body.request.message) || 'Upgrade requested.', 'success');
        }
    } catch (e) {
        tell('Request not sent: ' + String(e && e.message ? e.message : e), 'error');
    } finally {
        busy = false;
        await refresh();
    }
}

// The module publishes the names its own markup renders (the handler rule: a
// module that renders a handler string must publish that name itself).
window.chooseCareerPath = chooseCareerPath;
window.promoteCareerRole = promoteCareerRole;
window.requestStaffCareerUpgrade = requestStaffCareerUpgrade;



