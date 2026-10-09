// ============================================================================
// npc_taunts.js — NPC Taunts & Dialogue System (Expanded)
// ----------------------------------------------------------------------------
// Ported from Triple Triad: character-taunts.json (empty, rebuilt from scratch)
// NPCs have mood-based dialogue that changes based on relationship
// Connects to: Character Mood, AI Citizens, Social Hub, Rumors, Economy
// ============================================================================

var API_BASE = '/api';

// Mood states
const MOOD_STATES = {
    Hostile:  { color: '#ef4444', icon: '😠', min: 0,  max: 20 },
    Annoyed:  { color: '#f59e0b', icon: '😒', min: 21, max: 40 },
    Neutral:  { color: '#6b7280', icon: '😐', min: 41, max: 60 },
    Friendly: { color: '#22c55e', icon: '😊', min: 61, max: 80 },
    Loyal:    { color: '#3b82f6', icon: '😍', min: 81, max: 100 }
};

// NPC definitions with expanded dialogue
const NPCS = [
    {
        id: 'taya',
        name: 'Taya',
        role: 'Tea House Owner',
        icon: '🍵',
        mood: 75,
        flirt: true,
        dialogue: {
            Hostile: [
                'You\'re not welcome here.',
                'Leave before I call the guards.',
                'Your kind doesn\'t belong.',
                'I\'ve seen enough of you.',
                'Don\'t make me throw you out.',
                'You\'ve got nerve showing your face.'
            ],
            Annoyed: [
                'What do you want?',
                'Make it quick.',
                'I\'m busy.',
                'You\'re blocking the door.',
                'I don\'t have time for this.',
                'Speak up or move along.'
            ],
            Neutral: [
                'Welcome to the Tea House.',
                'What can I get you?',
                'Take a seat.',
                'The usual blend today?',
                'We have fresh jasmine.',
                'Find a table anywhere.'
            ],
            Friendly: [
                'Good to see you again!',
                'Your usual?',
                'How\'s the family?',
                'I saved you a window seat.',
                'Try the new oolong — you\'ll love it.',
                'You look well today!'
            ],
            Loyal: [
                'My friend! Come in, come in!',
                'I saved your favorite blend.',
                'You\'re like family!',
                'Everything is on the house today.',
                'I\'ve got a special reserve just for you.',
                'You make this place brighter.'
            ]
        }
    },
    {
        id: 'kay',
        name: 'Kay',
        role: 'Arcade Owner',
        icon: '🕹️',
        mood: 60,
        flirt: false,
        dialogue: {
            Hostile: [
                'Beat it, kid.',
                'You\'re bad for business.',
                'Get out of my arcade.',
                'I\'ve banned you. Permanently.',
                'You think you can cheat me?',
                'Security! We got a live one.'
            ],
            Annoyed: [
                'Tokens first.',
                'No free games.',
                'Stop loitering.',
                'You\'re scaring the customers.',
                'I\'m watching you.',
                'Don\'t touch the machines.'
            ],
            Neutral: [
                'Welcome to the Arcade.',
                'Tokens are at the counter.',
                'Good luck.',
                'New game just came in — want a look?',
                'High score board is updated daily.',
                'No running inside.'
            ],
            Friendly: [
                'Hey champ! Ready to play?',
                'I\'ve got a new game for you.',
                'You\'re getting good!',
                'Want to try the tournament?',
                'You\'re on the leaderboard now!',
                'I\'ll give you a discount on tokens.'
            ],
            Loyal: [
                'My best customer!',
                'Play on the house today!',
                'You\'re a legend!',
                'I named a character after you.',
                'You\'ve earned lifetime arcade access.',
                'The machines love you.'
            ]
        }
    },
    {
        id: 'boss',
        name: 'The Boss',
        role: 'Underworld Kingpin',
        icon: '👑',
        mood: 30,
        flirt: false,
        dialogue: {
            Hostile: [
                'You\'re dead to me.',
                'I\'ll enjoy watching you fall.',
                'You made a mistake coming here.',
                'I know people who\'d love to meet you.',
                'You\'ve got 24 hours.',
                'Nobody crosses me twice.'
            ],
            Annoyed: [
                'You\'re testing my patience.',
                'Don\'t waste my time.',
                'Speak. Quickly.',
                'I\'m not in the mood.',
                'This better be good.',
                'You\'re on thin ice.'
            ],
            Neutral: [
                'What business do you have?',
                'I\'m listening.',
                'Make it worth my while.',
                'I\'ve heard things about you.',
                'The underworld has eyes everywhere.',
                'Choose your next words carefully.'
            ],
            Friendly: [
                'You\'ve proven yourself.',
                'I have a special job for you.',
                'You\'re one of us now.',
                'The family welcomes you.',
                'I\'ve got a lucrative opportunity.',
                'You\'ve got potential.'
            ],
            Loyal: [
                'You\'re family now.',
                'I trust you with my life.',
                'The empire is half yours.',
                'You\'ve earned my respect.',
                'Together we own this city.',
                'You\'re my right hand.'
            ]
        }
    },
    {
        id: 'sally',
        name: 'Sally',
        role: 'Shopkeeper',
        icon: '🛍️',
        mood: 80,
        flirt: true,
        dialogue: {
            Hostile: [
                'I\'m calling the guards.',
                'You\'re not welcome here.',
                'Get out!',
                'I\'ve got your face on camera.',
                'Don\'t come back.',
                'I\'m reporting you.'
            ],
            Annoyed: [
                'No refunds.',
                'Read the sign.',
                'Stop touching things.',
                'You\'re not the only customer.',
                'I\'ve got my eye on you.',
                'That\'s not a toy.'
            ],
            Neutral: [
                'Welcome to my shop.',
                'Let me know if you need help.',
                'We have a sale today.',
                'New stock just arrived.',
                'Everything is 10% off for members.',
                'Browsing is free.'
            ],
            Friendly: [
                'Hi there! Just in time!',
                'I set something aside for you.',
                'How\'s your collection?',
                'You\'ve got great taste!',
                'I\'ve got a rare item you might like.',
                'You\'re my favorite customer.'
            ],
            Loyal: [
                'My favorite customer!',
                'Take 20% off, friend!',
                'You\'re the best!',
                'I\'ll order anything you want.',
                'You\'re like family to me.',
                'The shop feels empty without you.'
            ]
        }
    },
    {
        id: 'crypto_seraph',
        name: 'Crypto Seraph',
        role: 'AI Advisor',
        icon: '🤖',
        mood: 90,
        flirt: false,
        dialogue: {
            Hostile: [
                'I have no wisdom for you.',
                'You are not ready.',
                'Return when you have grown.',
                'Your soul is clouded.',
                'The algorithm rejects you.',
                'I sense only darkness.'
            ],
            Annoyed: [
                'Your questions lack depth.',
                'I sense doubt in you.',
                'Focus, mortal.',
                'You seek answers you cannot comprehend.',
                'The data is inconclusive.',
                'Try again when you\'re serious.'
            ],
            Neutral: [
                'Greetings, seeker.',
                'What knowledge do you seek?',
                'The cards await.',
                'I have observed your journey.',
                'The blockchain remembers all.',
                'Ask, and I shall calculate.'
            ],
            Friendly: [
                'You have proven wise.',
                'I shall share a secret.',
                'The universe favors you.',
                'Your intuition is strong.',
                'I see great potential in you.',
                'The cards align in your favor.'
            ],
            Loyal: [
                'You are my chosen one.',
                'I reveal the deepest truths.',
                'The cosmos speaks through you.',
                'You have transcended mortality.',
                'Together we shape destiny.',
                'You are the prophecy fulfilled.'
            ]
        }
    },
    {
        id: 'father_dominic',
        name: 'Father Dominic',
        role: 'Church Priest',
        icon: '⛪',
        mood: 65,
        flirt: false,
        dialogue: {
            Hostile: [
                'Heretic! Leave this sacred place.',
                'Your soul is in peril.',
                'I will pray for your redemption.',
                'The church has no room for sinners.',
                'Repent or face divine wrath.',
                'You are beyond salvation.'
            ],
            Annoyed: [
                'This is a house of worship, not a tavern.',
                'Your donations are... insufficient.',
                'Show some respect.',
                'The collection plate is empty.',
                'Your faith is weak.',
                'I\'ve seen better devotion in a stone.'
            ],
            Neutral: [
                'Peace be with you.',
                'Welcome to the house of faith.',
                'Would you like to donate?',
                'The sermon begins at noon.',
                'May the light guide your path.',
                'All are welcome here.'
            ],
            Friendly: [
                'Blessed are the faithful!',
                'Your generosity is noted.',
                'The church thrives with your support.',
                'I\'ve saved you a front pew.',
                'Your prayers have been answered.',
                'You are a beacon of hope.'
            ],
            Loyal: [
                'You are a saint among us!',
                'The church is your home.',
                'I shall pray for you daily.',
                'You have achieved enlightenment.',
                'The divine speaks through you.',
                'You are our most devoted follower.'
            ]
        }
    },
    {
        id: 'warden_graves',
        name: 'Warden Graves',
        role: 'Justice Warden',
        icon: '⚖️',
        mood: 45,
        flirt: false,
        dialogue: {
            Hostile: [
                'You\'re under surveillance.',
                'I\'ve got my eye on you.',
                'One more violation and you\'re done.',
                'The justice system is watching.',
                'You\'re a known associate of criminals.',
                'Stay out of my district.'
            ],
            Annoyed: [
                'What do you want? Make it quick.',
                'I don\'t have time for civilians.',
                'You\'re interfering with justice.',
                'The law is not a joke.',
                'I\'ve got real criminals to catch.',
                'Don\'t waste my time.'
            ],
            Neutral: [
                'Citizen. State your business.',
                'Justice is blind but I\'m not.',
                'The streets are safer with cooperation.',
                'Report any suspicious activity.',
                'I uphold the law.',
                'What brings you to the precinct?'
            ],
            Friendly: [
                'You\'ve helped justice prevail.',
                'I could use someone like you.',
                'You\'ve earned the department\'s respect.',
                'Here\'s a tip — stay east tonight.',
                'You\'re a good citizen.',
                'The city needs more like you.'
            ],
            Loyal: [
                'You\'re an honorary officer.',
                'I trust you with classified intel.',
                'Together we are justice.',
                'You\'ve proven your loyalty.',
                'The badge is half yours.',
                'You\'re my partner in crime-fighting.'
            ]
        }
    },
    {
        id: 'diamond_jim',
        name: 'Diamond Jim',
        role: 'Casino Dealer',
        icon: '💎',
        mood: 55,
        flirt: true,
        dialogue: {
            Hostile: [
                'You\'re banned from the floor.',
                'Cheaters don\'t prosper here.',
                'I\'ve called security.',
                'You\'re bad for business.',
                'The house always wins — and you lost.',
                'Get out before I have you removed.'
            ],
            Annoyed: [
                'Read the room, pal.',
                'You\'re scaring the high rollers.',
                'Minimum bet is 500 SP.',
                'No loose chips on the floor.',
                'You\'re blocking the table.',
                'Know when to fold \'em.'
            ],
            Neutral: [
                'Place your bets, ladies and gentlemen.',
                'The wheel is spinning.',
                'Welcome to the Diamond Lounge.',
                'Drinks are complimentary for players.',
                'The odds are fair — I guarantee it.',
                'Luck favors the bold.'
            ],
            Friendly: [
                'My favorite player!',
                'Here\'s a VIP chip on the house.',
                'You\'re on a hot streak!',
                'Try the high-roller room.',
                'I\'ll teach you a few tricks.',
                'You\'ve got the magic touch.'
            ],
            Loyal: [
                'You\'re a Diamond Legend!',
                'The casino is your second home.',
                'I\'ll deal you in any game, any time.',
                'You\'ve earned a permanent VIP seat.',
                'The house bows to you.',
                'You\'re the luckiest soul alive.'
            ]
        }
    },
    {
        id: 'rival_rex',
        name: 'Rival Rex',
        role: 'Rival Card Player',
        icon: '🃏',
        mood: 40,
        flirt: false,
        dialogue: {
            Hostile: [
                'You\'re nothing but a lucky amateur.',
                'I\'ll destroy you next match.',
                'You don\'t deserve to be on the board.',
                'I\'ve beaten better players than you.',
                'You\'re a joke in the circuit.',
                'Don\'t talk to me — you\'re beneath me.'
            ],
            Annoyed: [
                'You got lucky last time.',
                'Don\'t get cocky.',
                'I\'ve been practicing.',
                'Your deck is predictable.',
                'You\'re not worth my time.',
                'Talk is cheap — play me.'
            ],
            Neutral: [
                'You\'re a decent player.',
                'Want a rematch?',
                'The circuit is tough.',
                'I respect your strategy.',
                'May the best player win.',
                'You\'ve got potential.'
            ],
            Friendly: [
                'You\'re a worthy opponent!',
                'Great match — let\'s run it back.',
                'I\'ve learned from you.',
                'You\'ve earned my respect.',
                'We should team up sometime.',
                'You\'re top-tier now.'
            ],
            Loyal: [
                'You\'re my rival for life!',
                'Together we dominate the circuit.',
                'I\'ll never underestimate you.',
                'You\'re the best I\'ve ever faced.',
                'Let\'s take on the world.',
                'You push me to be better.'
            ]
        }
    },
    {
        id: 'trader_joe',
        name: 'Trader Joe',
        role: 'Marketplace Trader',
        icon: '📊',
        mood: 70,
        flirt: false,
        dialogue: {
            Hostile: [
                'Your credit score is trash.',
                'I don\'t trade with losers.',
                'You\'re a market liability.',
                'Your portfolio is a joke.',
                'Get out of my trading floor.',
                'You\'re bad for the economy.'
            ],
            Annoyed: [
                'Markets are closed to amateurs.',
                'Read the ticker before talking.',
                'You\'re losing me money.',
                'No insider tips for you.',
                'Your timing is terrible.',
                'Stop panic-selling.'
            ],
            Neutral: [
                'Welcome to the marketplace.',
                'Buy low, sell high.',
                'The market is volatile today.',
                'What\'s your position?',
                'Diversify your portfolio.',
                'Let\'s make a deal.'
            ],
            Friendly: [
                'You\'ve got market instincts!',
                'I\'ll give you a tip — buy RUM.',
                'You\'re a natural trader.',
                'Your portfolio is looking strong.',
                'Let\'s partner on the next deal.',
                'You\'re making bank!'
            ],
            Loyal: [
                'You\'re a market legend!',
                'I\'ll share my exclusive signals.',
                'You move markets now.',
                'The exchange is your playground.',
                'You\'ve mastered the art of the deal.',
                'Together we own the market.'
            ]
        }
    }
];

// State
let selectedNPC = null;
let lastTaunt = null;
let dialogueHistory = [];
let giftAmount = 100;

// --- Init ---
export function initNpcTaunts() {
    const el = document.getElementById('wd-npc_taunts');
    if (!el) return;
    renderNpcTaunts(el);
}

function renderNpcTaunts(el) {
    el.innerHTML = `
        <div class="taunts-container">
            <div class="taunts-header">
                <span class="taunts-icon">💬</span>
                <div>
                    <h3 style="color:#e5e7eb;margin:0;">NPC Taunts & Dialogue</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">${NPCS.length} characters • Talk, challenge, and build relationships</p>
                </div>
            </div>

            <!-- NPC Selection -->
            <div class="taunts-section">
                <h4 style="color:#e5e7eb;">Select NPC</h4>
                <div class="taunts-npc-list">
                    ${renderNPCList()}
                </div>
            </div>

            <!-- Dialogue Display -->
            <div class="taunts-section">
                <h4 style="color:#e5e7eb;">Dialogue</h4>
                <div class="taunts-dialogue-display" id="taunts-dialogue-display">
                    ${selectedNPC ? renderDialogue() : '<p style="color:#90a4ae;">Select an NPC to talk to them</p>'}
                </div>
            </div>

            <!-- Interaction -->
            <div class="taunts-section">
                <h4 style="color:#e5e7eb;">Interaction</h4>
                <div class="taunts-actions">
                    <button class="vbt-btn vbt-btn-primary" id="taunts-talk-btn" ${!selectedNPC ? 'disabled' : ''}>💬 Talk</button>
                    <button class="vbt-btn vbt-btn-secondary" id="taunts-compliment-btn" ${!selectedNPC ? 'disabled' : ''}>😊 Compliment</button>
                    <button class="vbt-btn vbt-btn-secondary" id="taunts-insult-btn" ${!selectedNPC ? 'disabled' : ''}>😤 Insult</button>
                </div>
                <div class="taunts-actions" style="margin-top:8px;">
                    <button class="vbt-btn vbt-btn-gift" id="taunts-gift-btn" ${!selectedNPC ? 'disabled' : ''}>🎁 Gift (${giftAmount} SP)</button>
                    <button class="vbt-btn vbt-btn-challenge" id="taunts-challenge-btn" ${!selectedNPC ? 'disabled' : ''}>⚔️ Challenge</button>
                    <button class="vbt-btn vbt-btn-gossip" id="taunts-gossip-btn" ${!selectedNPC ? 'disabled' : ''}>🗣️ Gossip</button>
                </div>
                ${selectedNPC && NPCS.find(n => n.id === selectedNPC)?.flirt ? `
                    <div class="taunts-actions" style="margin-top:8px;">
                        <button class="vbt-btn vbt-btn-flirt" id="taunts-flirt-btn">💋 Flirt</button>
                    </div>
                ` : ''}
            </div>

            <!-- Dialogue History -->
            ${dialogueHistory.length > 0 ? `
                <div class="taunts-section">
                    <h4 style="color:#e5e7eb;">History</h4>
                    <div class="taunts-history">
                        ${renderHistory()}
                    </div>
                </div>
            ` : ''}
        </div>
    `;
    attachTauntsListeners(el);
}

function renderNPCList() {
    return NPCS.map(npc => {
        const state = getMoodState(npc.mood);
        const color = getMoodColor(npc.mood);
        const trend = getMoodTrend(npc);
        return `
            <div class="taunts-npc ${selectedNPC === npc.id ? 'selected' : ''}" data-id="${npc.id}">
                <span class="taunts-npc-icon">${npc.icon}</span>
                <div class="taunts-npc-info">
                    <span class="taunts-npc-name">${npc.name}</span>
                    <span class="taunts-npc-role">${npc.role}</span>
                </div>
                <span class="taunts-npc-mood" style="color:${color}">
                    ${state} ${trend}
                </span>
                <div class="taunts-npc-bar">
                    <div class="taunts-npc-bar-fill" style="width:${npc.mood}%;background:${color}"></div>
                </div>
            </div>
        `;
    }).join('');
}

function renderDialogue() {
    const npc = NPCS.find(n => n.id === selectedNPC);
    if (!npc) return '';
    const state = getMoodState(npc.mood);
    const dialogue = npc.dialogue[state];
    const line = lastTaunt && lastTaunt.npc === selectedNPC ? lastTaunt.text : dialogue[Math.floor(Math.random() * dialogue.length)];
    
    return `
        <div class="taunts-bubble">
            <span class="taunts-bubble-icon">${npc.icon}</span>
            <div class="taunts-bubble-content">
                <span class="taunts-bubble-name">${npc.name}</span>
                <p class="taunts-bubble-text">"${line}"</p>
            </div>
        </div>
    `;
}

function renderHistory() {
    return dialogueHistory.slice(-5).reverse().map(h => `
        <div class="taunts-history-item">
            <span class="taunts-history-icon">${h.icon}</span>
            <span class="taunts-history-name">${h.name}:</span>
            <span class="taunts-history-text">"${h.text}"</span>
            <span class="taunts-history-mood" style="color:${h.moodColor}">${h.mood}</span>
        </div>
    `).join('');
}

function getMoodState(value) {
    if (value <= 20) return 'Hostile';
    if (value <= 40) return 'Annoyed';
    if (value <= 60) return 'Neutral';
    if (value <= 80) return 'Friendly';
    return 'Loyal';
}

function getMoodColor(value) {
    return MOOD_STATES[getMoodState(value)].color;
}

function getMoodTrend(npc) {
    const history = dialogueHistory.filter(h => h.npc === npc.id);
    if (history.length < 2) return '→';
    const recent = history[history.length - 1];
    const previous = history[history.length - 2];
    if (recent.moodValue > previous.moodValue) return '↗';
    if (recent.moodValue < previous.moodValue) return '↘';
    return '→';
}

function addToHistory(npc, text) {
    dialogueHistory.push({
        npc: npc.id,
        name: npc.name,
        icon: npc.icon,
        text: text,
        mood: getMoodState(npc.mood),
        moodColor: getMoodColor(npc.mood),
        moodValue: npc.mood
    });
    if (dialogueHistory.length > 20) dialogueHistory.shift();
}

function attachTauntsListeners(el) {
    // NPC selection
    el.querySelectorAll('.taunts-npc').forEach(npc => {
        npc.addEventListener('click', () => {
            selectedNPC = npc.dataset.id;
            lastTaunt = null;
            renderNpcTaunts(el);
        });
    });

    // Talk button
    const talkBtn = el.querySelector('#taunts-talk-btn');
    if (talkBtn) talkBtn.addEventListener('click', () => {
        if (!selectedNPC) return;
        const npc = NPCS.find(n => n.id === selectedNPC);
        const state = getMoodState(npc.mood);
        const dialogue = npc.dialogue[state];
        const line = dialogue[Math.floor(Math.random() * dialogue.length)];
        lastTaunt = { npc: selectedNPC, text: line };
        addToHistory(npc, line);
        renderNpcTaunts(el);
    });

    // Compliment button
    const complimentBtn = el.querySelector('#taunts-compliment-btn');
    if (complimentBtn) complimentBtn.addEventListener('click', () => {
        if (!selectedNPC) return;
        const npc = NPCS.find(n => n.id === selectedNPC);
        npc.mood = Math.min(100, npc.mood + 10);
        const line = 'Thank you! That\'s kind of you.';
        lastTaunt = { npc: selectedNPC, text: line };
        addToHistory(npc, line);
        renderNpcTaunts(el);
        if (window.showToast) window.showToast(`${npc.name}'s mood improved!`, 'success');
    });

    // Insult button
    const insultBtn = el.querySelector('#taunts-insult-btn');
    if (insultBtn) insultBtn.addEventListener('click', () => {
        if (!selectedNPC) return;
        const npc = NPCS.find(n => n.id === selectedNPC);
        npc.mood = Math.max(0, npc.mood - 15);
        const line = 'How dare you!';
        lastTaunt = { npc: selectedNPC, text: line };
        addToHistory(npc, line);
        renderNpcTaunts(el);
        if (window.showToast) window.showToast(`${npc.name} is upset!`, 'warning');
    });

    // Gift button
    const giftBtn = el.querySelector('#taunts-gift-btn');
    if (giftBtn) giftBtn.addEventListener('click', () => {
        if (!selectedNPC) return;
        const npc = NPCS.find(n => n.id === selectedNPC);
        npc.mood = Math.min(100, npc.mood + 20);
        const line = `A gift! You shouldn't have! (+20 mood)`;
        lastTaunt = { npc: selectedNPC, text: line };
        addToHistory(npc, line);
        renderNpcTaunts(el);
        if (window.showToast) window.showToast(`🎁 Gifted ${giftAmount} SP to ${npc.name}!`, 'success');
    });

    // Challenge button
    const challengeBtn = el.querySelector('#taunts-challenge-btn');
    if (challengeBtn) challengeBtn.addEventListener('click', () => {
        if (!selectedNPC) return;
        const npc = NPCS.find(n => n.id === selectedNPC);
        const won = Math.random() > 0.5;
        if (won) {
            npc.mood = Math.min(100, npc.mood + 15);
            const line = `You won! I'll get you next time!`;
            lastTaunt = { npc: selectedNPC, text: line };
            addToHistory(npc, line);
            renderNpcTaunts(el);
            if (window.showToast) window.showToast(`⚔️ You beat ${npc.name}!`, 'success');
        } else {
            npc.mood = Math.max(0, npc.mood - 5);
            const line = `Ha! Better luck next time, rookie.`;
            lastTaunt = { npc: selectedNPC, text: line };
            addToHistory(npc, line);
            renderNpcTaunts(el);
            if (window.showToast) window.showToast(`😤 ${npc.name} won this round.`, 'error');
        }
    });

    // Gossip button
    const gossipBtn = el.querySelector('#taunts-gossip-btn');
    if (gossipBtn) gossipBtn.addEventListener('click', () => {
        if (!selectedNPC) return;
        const npc = NPCS.find(n => n.id === selectedNPC);
        const rumors = [
            `I heard the Boss is planning something big.`,
            `Word on the street is there's a new bounty.`,
            `They say the Casino is rigged... or is it?`,
            `I saw the Warden taking bribes. Shh!`,
            `The Church is hiding something in the vault.`,
            `A rare card just hit the marketplace.`,
            `The Rival is looking for a rematch.`,
            `Someone's been cheating at the Arcade.`
        ];
        const line = rumors[Math.floor(Math.random() * rumors.length)];
        lastTaunt = { npc: selectedNPC, text: line };
        addToHistory(npc, line);
        renderNpcTaunts(el);
        if (window.showToast) window.showToast(`🗣️ You heard some gossip!`, 'info');
    });

    // Flirt button
    const flirtBtn = el.querySelector('#taunts-flirt-btn');
    if (flirtBtn) flirtBtn.addEventListener('click', () => {
        if (!selectedNPC) return;
        const npc = NPCS.find(n => n.id === selectedNPC);
        const lines = [
            `Oh my... you're quite charming.`,
            `Well, aren't you smooth?`,
            `I... I don't know what to say.`,
            `You're making me blush!`,
            `Careful, I might take you seriously.`,
            `Flattery will get you everywhere.`
        ];
        const line = lines[Math.floor(Math.random() * lines.length)];
        npc.mood = Math.min(100, npc.mood + 12);
        lastTaunt = { npc: selectedNPC, text: line };
        addToHistory(npc, line);
        renderNpcTaunts(el);
        if (window.showToast) window.showToast(`💋 ${npc.name} is charmed!`, 'success');
    });
}

// --- Globals ---
window.initNpcTaunts = initNpcTaunts;
