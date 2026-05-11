// Public/js/config.js

// --- Global Deployment Configuration ---
export const CONFIG = (() => {
    const isLocal = window.location.hostname === "localhost" || window.location.hostname === "127.0.0.1";
    const backendHost = window.location.host; // Dynamically uses current host (localhost:8082 or Render URL)
    return {
        IS_LOCAL: isLocal,
        BACKEND_URL: backendHost,
        API_BASE: (window.location.protocol === "https:" ? "https://" : "http://") + backendHost,
        // Production CDN: Link to the Public folder in the deploy branch
        ASSET_URL: isLocal ? "/" : "https://raw.githubusercontent.com/slapkarnts/VOiconomy-faucet/deploy/Public/",
        WC_PROJECT_ID: document.querySelector('meta[name="walletconnect-project-id"]')?.content || 'your_walletconnect_project_id', // Set this in index.html or replace with a real project ID
        VOI_CHAIN_ID: 'algorand:wGHE2Pwd1-YdV4EuJFy9u6C24-L-2B05',
        ALGO_CHAIN_ID: 'algorand:mainnet-v1.0',
        VAULT_ADDRESS: null,                      // Dynamic: Synced from server on connect
        VBV_ASSET_ID: null,                       // Dynamic: Synced from server on connect
        AVOI_ASSET_ID: null                       // Dynamic: Synced from server on connect
    };
})();