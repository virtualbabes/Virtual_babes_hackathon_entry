// ============================================================================
// PILLAR 13: RIVALRY ENGINE - Frontend Module
// Handles rival requests, accept/reject flow, PvP challenges, and
// faction shop browsing/purchasing. Integrates with Network API and UI panel.
// ============================================================================

const RivalryEngine = (() => {
    let state = {
        activeRivals: [],
        pendingInvitations: [],
        careerXP: 0,
        level: 0,
        faction: '',
        shopItems: [],
        isShopOpen: false,
        isLoading: false,
    };

    // --- API Calls ---

    async function requestRival(targetWallet) {
        try {
            const result = await Network.apiPost('/api/rivalry/request', { target_wallet: targetWallet });
            if (result.success) {
                UI.showNotification('Rival request sent!', 'success');
            } else {
                UI.showNotification(result.error || 'Failed to send rival request.', 'error');
            }
            return result;
        } catch (err) {
            console.error('[RIVALRY] Request rival failed:', err);
            UI.showNotification('Network error sending rival request.', 'error');
        }
    }

    async function acceptRival(rivalWallet) {
        try {
            const result = await Network.apiPost('/api/rivalry/accept', { rival_wallet: rivalWallet, status: 'accepted' });
            if (result.success) {
                state.pendingInvitations = state.pendingInvitations.filter(r => r.wallet !== rivalWallet);
                state.activeRivals.push({ wallet: rivalWallet, name: result.name || rivalWallet.slice(0, 8) + '...', level: result.level || 1 });
                UI.renderRivalPanel();
                UI.showNotification('Rivalry accepted!', 'success');
            } else {
                UI.showNotification(result.error || 'Failed to accept rival.', 'error');
            }
            return result;
        } catch (err) {
            console.error('[RIVALRY] Accept rival failed:', err);
        }
    }

    async function declineRival(rivalWallet) {
        try {
            const result = await Network.apiPost('/api/rivalry/decline', { rival_wallet: rivalWallet });
            if (result.success) {
                state.pendingInvitations = state.pendingInvitations.filter(r => r.wallet !== rivalWallet);
                UI.renderRivalPanel();
                UI.showNotification('Rival request declined.', 'info');
            } else {
                UI.showNotification(result.error || 'Failed to decline rival.', 'error');
            }
        } catch (err) {
            console.error('[RIVALRY] Decline rival failed:', err);
        }
    }

    async function challengeRival(rivalWallet) {
        try {
            const result = await Network.apiPost('/api/rivalry/challenge', { rival_wallet: rivalWallet });
            if (result.success) {
                UI.showNotification(`Challenge sent to ${rivalWallet}!`, 'success');
            } else {
                UI.showNotification(result.error || 'Challenge failed.', 'error');
            }
        } catch (err) {
            console.error('[RIVALRY] Challenge failed:', err);
        }
    }

    async function getFactionShop(faction) {
        try {
            const result = await Network.apiGet(`/api/faction/shop/${faction.toUpperCase()}`);
            if (result.success) {
                state.faction = faction.toLowerCase();
                state.shopItems = result.items || [];
                state.isShopOpen = true;
                UI.renderFactionShop();
            } else {
                UI.showNotification(result.error || 'Failed to load shop.', 'error');
            }
        } catch (err) {
            console.error('[RIVALRY] Shop load failed:', err);
            UI.showNotification('Network error loading faction shop.', 'error');
        }
    }

    async function buyFactionItem(itemName) {
        if (state.isLoading) return;
        state.isLoading = true;
        try {
            const result = await Network.apiPost('/api/faction/shop/buy', { faction: state.faction, item_name: itemName });
            if (result.success) {
                UI.showNotification(`Purchased ${itemName}!`, 'success');
                state.shopItems = state.shopItems.filter(i => i.id !== itemName);
                UI.renderFactionShop();
            } else {
                UI.showNotification(result.error || 'Purchase failed.', 'error');
            }
        } catch (err) {
            console.error('[RIVALRY] Buy failed:', err);
            UI.showNotification('Network error purchasing item.', 'error');
        } finally {
            state.isLoading = false;
        }
    }

    async function syncCareerProgress() {
        try {
            const result = await Network.apiGet('/api/career/progress');
            if (result.success) {
                state.careerXP = result.data?.total_xp || 0;
                state.level = result.data?.level || 0;
                UI.updateCareerBar(result.data);
            }
        } catch (err) {
            console.error('[RIVALRY] Career sync failed:', err);
        }
    }

    function getState() { return state; }

    return {
        requestRival, acceptRival, declineRival, challengeRival,
        getFactionShop, buyFactionItem, syncCareerProgress, getState,
    };
})();
