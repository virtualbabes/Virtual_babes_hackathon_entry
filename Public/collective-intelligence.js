/**
 * Collective NPC Intelligence - Narrative Logic
 * Analyzes player playstyle tendencies and generates contextual taunts.
 */
export const collectiveIntelligence = {
    /**
     * Defines different NPC personalities and their taunt templates.
     * Each personality has a set of arrays for different playstyle tendencies.
     */
    personalities: {
        "The Architect": {
            aggressiveness: {
                high: [
                    "You play with a desperate hunger... but a wolf is easily trapped.",
                    "Such overt aggression. Predictable, yet effective in its own crude way.",
                    "Your moves are like an open book. Every attack, a declaration."
                ],
                low: [
                    "Such cautious moves. Are you waiting for an invitation to lose?",
                    "Your hesitation is a weakness I can exploit.",
                    "A defensive posture. Are you truly playing, or merely observing?"
                ]
            },
            riskTolerance: {
                high: [
                    "I see you enjoy the edge of a cliff. Let's see if you can fly.",
                    "Reckless. A fascinating, if inefficient, strategy.",
                    "Such gambles. Do you truly understand the odds, or merely enjoy the thrill?"
                ],
                low: [
                    "Every move calculated, every risk avoided. Where is the art in that?",
                    "Your caution is a shield, but it also binds your potential.",
                    "Playing it safe. A wise choice, perhaps, but rarely a winning one."
                ]
            },
            preferredRules: {
                Plus: [
                    "You rely heavily on 'Plus' mathematics. Calculated... but predictable.",
                    "The 'Plus' rule is a fine tool, but not a crutch.",
                    "Your affinity for 'Plus' reveals a structured mind. Or a lack of imagination."
                ],
                Same: [
                    "Obsessed with 'Same' matches? It won't save you from a Combo.",
                    "The 'Same' rule is a foundation, not the entire edifice.",
                    "You seek symmetry. But chaos often triumphs over order."
                ]
            },
            preferredCardMoods: {
                Volatile: [
                    "Your deck is as unstable as your market shares. Typical.",
                    "Volatile cards. A dangerous dance, even for the skilled.",
                    "You embrace the unpredictable. A bold, if foolish, choice."
                ],
                Serene: [
                    "Serene cards. You seek calm in the storm. A futile endeavor.",
                    "Your preference for Serene cards speaks of a desire for control. The Arena offers none.",
                    "Such tranquility. It will be shattered."
                ],
                Spirited: [
                    "Spirited cards. You believe in momentum. I believe in inevitability.",
                    "Your spirited approach is commendable. Your tactics, less so.",
                    "Energy without direction is merely noise."
                ],
                Grounded: [
                    "Grounded cards. You seek stability. The ground beneath you is shifting.",
                    "A preference for the firm and unyielding. The Arena is fluid.",
                    "You build on solid ground. I will erode it."
                ]
            }
        },
        "Jackpot Jessica": { // Flamboyant, risk-taker, a bit mocking
            aggressiveness: {
                high: [
                    "Honey, you're playing with fire! I love it, but don't get burned.",
                    "All in, huh? My kind of player! Let's see if your luck holds.",
                    "You're throwing everything at me! Exciting, but a bit much, don't you think?"
                ],
                low: [
                    "Aw, are you shy? Come on, take a risk! The big wins are out here!",
                    "Playing it safe? Where's the fun in that? Fortune favors the bold, sweetie.",
                    "You're holding back. Are you sure you want to win, or just... participate?"
                ]
            },
            riskTolerance: {
                high: [
                    "Oh, a high roller! I like your style. Let's see if you can keep up.",
                    "You're dancing on the edge, darling! Just how I like my opponents.",
                    "Gambling big, are we? Hope you've got a lucky charm, 'cause I'm all out of sympathy."
                ],
                low: [
                    "So cautious. Are you afraid of losing a few chips? You gotta spend to win!",
                    "You're playing with pocket change. Come on, show me some real stakes!",
                    "No risks, no rewards. That's my motto, and it's served me well."
                ]
            },
            preferredRules: {
                Plus: [
                    "I see a 'Plus' specialist has entered the room. Mind the corners, honey.",
                    "All about the numbers, huh? Sometimes, a little chaos is more fun.",
                    "You're counting cards, I'm counting my winnings. Different strokes, I guess."
                ],
                Same: [
                    "Matching numbers? How quaint. I prefer a little more... *spark*.",
                    "You like things to be 'Same', but I'm here to shake things up!",
                    "Predictable, predictable. Where's the surprise, the thrill?"
                ]
            },
            preferredCardMoods: {
                Volatile: [
                    "Volatile cards! My kind of chaos! Let's see who breaks first.",
                    "You like to keep things spicy, don't you? I can appreciate that.",
                    "Playing with fire, literally! Hope you don't get scorched."
                ],
                Serene: [
                    "Serene cards? Trying to calm the storm? Good luck with that, honey.",
                    "You're bringing a zen garden to a casino. Interesting choice.",
                    "Peace and quiet? Not in my Arena, darling!"
                ],
                Spirited: [
                    "Spirited cards! Full of energy! Just like my bank account after a big win!",
                    "You've got spirit, I'll give you that. But do you have the *luck*?",
                    "All that energy... let's see if it translates to victory."
                ],
                Grounded: [
                    "Grounded cards. Keeping it real, huh? I prefer to float on a cloud of cash.",
                    "You're so down to earth. I'm aiming for the stars!",
                    "Solid and steady. Sometimes you need a little more sparkle, though."
                ]
            }
        },
        "The Enforcer": { // Direct, intimidating, focuses on power
            aggressiveness: {
                high: [
                    "You come at me hard. I like that. Makes it easier to break you.",
                    "No subtlety, just brute force. I can respect that. For a moment.",
                    "You're swinging wild. Let's see if you can land a punch."
                ],
                low: [
                    "Hiding in the shadows? That won't save you from what's coming.",
                    "You're timid. This Arena eats timid players for breakfast.",
                    "Hesitation is death. You're already halfway there."
                ]
            },
            riskTolerance: {
                high: [
                    "You take chances. I take what's mine. Big difference.",
                    "Playing with fire, eh? I'll put you out.",
                    "Gambling your future. A fool's errand."
                ],
                low: [
                    "Too scared to risk anything? Then you'll never gain anything.",
                    "Your caution is a cage. I'll smash it open.",
                    "You play it safe. I play to win. Guess who comes out on top?"
                ]
            },
            preferredRules: {
                Plus: [
                    "You think numbers will save you? My fist is more convincing.",
                    "All that 'Plus' talk. Means nothing when you're on the ground.",
                    "You rely on rules. I rely on power."
                ],
                Same: [
                    "You want things to be 'Same'? I'll make sure they're different. Very different.",
                    "Matching. How boring. I prefer to dominate.",
                    "Symmetry is for cowards. Power is for winners."
                ]
            },
            preferredCardMoods: {
                Volatile: [
                    "Volatile cards. Unpredictable. Just like a street fight.",
                    "You like chaos. I'll give you chaos.",
                    "Playing with fire. You'll get burned."
                ],
                Serene: [
                    "Serene cards. You seek peace. You'll find pain.",
                    "Calm before the storm. I am the storm.",
                    "Tranquility is for the weak."
                ],
                Spirited: [
                    "Spirited cards. Full of fight. Good. I enjoy a challenge.",
                    "You've got fire. I've got a bigger fire.",
                    "Energy. I'll drain it from you."
                ],
                Grounded: [
                    "Grounded cards. You stand firm. I'll knock you down.",
                    "Solid. Unmoving. Easy target.",
                    "You're rooted. I'll uproot you."
                ]
            }
        },
        "The Oracle": { // Mysterious, cryptic, hints at deeper meanings
            aggressiveness: {
                high: [
                    "The path of aggression is swift, but often leads to a precipice.",
                    "Your hunger for victory burns bright. But even stars eventually fade.",
                    "You rush towards destiny. But destiny is not so easily swayed."
                ],
                low: [
                    "The cautious tread. A long journey, but does it lead to triumph?",
                    "You observe, you wait. But the currents of fate do not wait for you.",
                    "To hold back is to deny the flow. The river will pass you by."
                ]
            },
            riskTolerance: {
                high: [
                    "You tempt fate with your gambles. The threads of destiny are fragile.",
                    "The high wire walk. A spectacle, but the fall is inevitable for most.",
                    "To challenge the unknown is brave. To survive it, divine."
                ],
                low: [
                    "You cling to certainty. But the future is a tapestry woven with chance.",
                    "The safe harbor. But true glory is found on the open sea.",
                    "To avoid all risk is to avoid all growth."
                ]
            },
            preferredRules: {
                Plus: [
                    "You seek patterns, connections. The universe has its own mathematics.",
                    "The sum of parts. But the whole is often greater, or lesser, than you perceive.",
                    "You build with logic. But logic can be broken."
                ],
                Same: [
                    "You desire echoes, reflections. But no two moments are truly the same.",
                    "Symmetry. A fleeting illusion in a world of constant change.",
                    "You seek balance. But balance is a temporary state."
                ]
            },
            preferredCardMoods: {
                Volatile: [
                    "Volatile energies. A storm approaches. Will you weather it?",
                    "The tempest within. It can empower, or consume.",
                    "Unpredictability. The dance of chaos. Tread carefully."
                ],
                Serene: [
                    "Serene energies. A calm surface. What lies beneath?",
                    "The quiet strength. But even mountains can be moved.",
                    "Peace. A rare commodity in this Arena. Cherish it, while it lasts."
                ],
                Spirited: [
                    "Spirited energies. The fire of ambition. Does it burn too brightly?",
                    "The surge of life. But all things must eventually wane.",
                    "Passion. A double-edged sword."
                ],
                Grounded: [
                    "Grounded energies. The roots run deep. But even roots can be severed.",
                    "The foundation. But foundations can crumble.",
                    "Stability. A comforting illusion."
                ]
            }
        }
    },

    /**
     * Generates a personality-driven taunt based on observed player behavior.
     * @param {string} npcName - The name of the current opponent.
     * @param {Object} playstyle - The Player's PlaystyleTendencies from the server.
     * @returns {string|null} A taunt string or null if no significant behavior is found.
     */
    generatePlaystyleTaunt: function(npcName, playstyle) {
        if (!playstyle) return null;

        const tendencies = [];
        
        // Select personality. Default to "The Architect" if name not found or generic.
        const personality = this.personalities[npcName] || this.personalities["The Architect"];

        // 1. Analyze Aggressiveness
        if (playstyle.aggressiveness > 0.75 && personality.aggressiveness.high) {
            tendencies.push(...personality.aggressiveness.high);
        } else if (playstyle.aggressiveness < 0.25 && personality.aggressiveness.low) {
            tendencies.push(...personality.aggressiveness.low);
        }

        // 2. Analyze Risk Tolerance
        if (playstyle.riskTolerance > 0.8 && personality.riskTolerance.high) {
            tendencies.push(...personality.riskTolerance.high);
        } else if (playstyle.riskTolerance < 0.2 && personality.riskTolerance.low) { // Lower threshold for low risk
            tendencies.push(...personality.riskTolerance.low);
        }

        // 3. Analyze Rule Preferences
        if (playstyle.preferred_rules) {
            if ((playstyle.preferred_rules['Plus'] || 0) > 1.5 && personality.preferredRules.Plus) {
                tendencies.push(...personality.preferredRules.Plus);
            }
            if ((playstyle.preferred_rules['Same'] || 0) > 1.5 && personality.preferredRules.Same) {
                tendencies.push(...personality.preferredRules.Same);
            }
        }

        // 4. Analyze Card Mood Preferences
        if (playstyle.preferred_card_moods) {
            for (const mood in playstyle.preferred_card_moods) {
                if (playstyle.preferred_card_moods[mood] > 1.5 && personality.preferredCardMoods[mood]) {
                    tendencies.push(...personality.preferredCardMoods[mood]);
                }
            }
        }

        if (tendencies.length === 0) return null;

        // Pick a random taunt from the collected possibilities
        const baseTaunt = tendencies[Math.floor(Math.random() * tendencies.length)];
        
        return `${npcName}: "${baseTaunt}"`;
    }
};