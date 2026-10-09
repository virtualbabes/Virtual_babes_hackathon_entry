// ============================================================================
// daily_challenges.js — Daily and weekly retention challenges
// ============================================================================

(function () {
    'use strict';
    var API_BASE = '/api';
    let overlayEl = null;
    let dailyChallenges = [];
    let weeklyChallenges = [];

    function init() {
        if (overlayEl) return;
        overlayEl = document.getElementById('daily-challenges-overlay');
        if (!overlayEl) return;
        overlayEl.className = 'overlay daily-challenges-overlay';
        overlayEl.style.display = 'none';

        overlayEl.innerHTML = `
            <div class="daily-challenges-panel">
                <button class="daily-challenges-close" onclick="window.closeDailyChallenges()">✕</button>
                <h2>🎯 Daily Challenges</h2>
                <div class="challenges-timer">Resets in: <span id="daily-timer">23:59:59</span></div>
                <div id="daily-list" class="challenges-list"></div>
                
                <h2>📅 Weekly Challenges</h2>
                <div class="challenges-timer">Resets in: <span id="weekly-timer">6d 23:59:59</span></div>
                <div id="weekly-list" class="challenges-list"></div>
            </div>`;

        loadChallenges();
    }

    function loadChallenges() {
        // Generate daily challenges
        dailyChallenges = [
            { id: 'd1', name: 'Win 3 battles', reward: 200, progress: 0, target: 3 },
            { id: 'd2', name: 'Claim church income', reward: 100, progress: 0, target: 1 },
            { id: 'd3', name: 'Buy something from market', reward: 150, progress: 0, target: 1 },
        ];

        // Generate weekly challenges
        weeklyChallenges = [
            { id: 'w1', name: 'Win 20 battles', reward: 2000, progress: 0, target: 20, bonus: 'rare_pet_egg' },
            { id: 'w2', name: 'Earn 5,000 VBV from churches', reward: 3000, progress: 0, target: 5000, bonus: 'cosmetic' },
            { id: 'w3', name: 'Reach top 100 leaderboard', reward: 5000, progress: 0, target: 100, bonus: 'title' },
        ];

        renderChallenges();
        startTimers();
    }

    function renderChallenges() {
        const dailyEl = document.getElementById('daily-list');
        const weeklyEl = document.getElementById('weekly-list');

        if (dailyEl) {
            dailyEl.innerHTML = dailyChallenges.map(c => renderChallenge(c)).join('');
        }
        if (weeklyEl) {
            weeklyEl.innerHTML = weeklyChallenges.map(c => renderChallenge(c)).join('');
        }
    }

    function renderChallenge(c) {
        const isComplete = c.progress >= c.target;
        const progress = Math.min(100, (c.progress / c.target) * 100);
        
        return `
        <div class="challenge-card ${isComplete ? 'complete' : ''}">
            <div class="challenge-info">
                <span class="challenge-name">${c?.name}</span>
                <span class="challenge-reward">+${c.reward} VBV${c.bonus ? ' +' : ''}</span>
            </div>
            <div class="challenge-progress">
                <div class="challenge-progress-bar">
                    <div class="challenge-progress-fill" style="width: ${progress}%"></div>
                </div>
                <span class="challenge-progress-text">${c.progress}/${c.target}</span>
            </div>
            ${isComplete ? '<button class="vbt-btn vbt-btn-primary claim-btn">Claim</button>' : ''}
        </div>`;
    }

    function startTimers() {
        // Daily timer
        let dailySeconds = 24 * 3600;
        setInterval(() => {
            dailySeconds--;
            if (dailySeconds <= 0) dailySeconds = 24 * 3600;
            const el = document.getElementById('daily-timer');
            if (el) el.textContent = formatTime(dailySeconds);
        }, 1000);

        // Weekly timer
        let weeklySeconds = 7 * 24 * 3600;
        setInterval(() => {
            weeklySeconds--;
            if (weeklySeconds <= 0) weeklySeconds = 7 * 24 * 3600;
            const el = document.getElementById('weekly-timer');
            if (el) el.textContent = formatTime(weeklySeconds);
        }, 1000);
    }

    function formatTime(seconds) {
        const d = Math.floor(seconds / (24 * 3600));
        const h = Math.floor((seconds % (24 * 3600)) / 3600);
        const m = Math.floor((seconds % 3600) / 60);
        const s = seconds % 60;
        return d > 0 ? `${d}d ${h.toString().padStart(2,'0')}:${m.toString().padStart(2,'0')}:${s.toString().padStart(2,'0')}` : `${h.toString().padStart(2,'0')}:${m.toString().padStart(2,'0')}:${s.toString().padStart(2,'0')}`;
    }

    window.openDailyChallenges = function () {
        init();
        overlayEl.style.display = 'flex';
    };

    window.closeDailyChallenges = function () {
        if (overlayEl) overlayEl.style.display = 'none';
    };

    if (document.getElementById('daily-challenges-overlay')) init();
})();
