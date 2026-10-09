//go:build !js && !wasm

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
)

// ============================================================================
// THE PUBLIC VIEW OF THE NETWORK REGISTRY
//
// AUTHORITY (binding). The network registry is admin-written; the SECRET material
// inside it is ENVIRONMENT-owned:
//
//	ALGOD_TOKEN_VOI / ALGOD_TOKEN_ALGO -> NetworkConfig.AlgodToken (threaded at boot, server.go)
//	IPFS_API_KEY                       -> NetworkConfig.IPFSAPIKey (threaded at boot, server.go)
//
// A secret must never leave the process, for two reasons that are both live:
//
//  1. `lobby_update` broadcasts the whole registry to EVERY connected client
//     (Lobby.getLobbyUpdateMsgLocked).
//  2. saveNetworkConfigs() persists it to networks.json, which is GIT-TRACKED.
//
// This file is the ONE owner of the projection both doors use. A field added to
// NetworkConfig is NOT public until it is added to NetworkConfigPublic, and
// assertNetworkRegistryRedaction() fails the boot if a served key ever looks like a
// secret — so a leak cannot be introduced by a later edit that forgets this file.
// ============================================================================

// NetworkConfigPublic is the client-safe projection of NetworkConfig: everything an
// operator UI needs to SEE, and no secret material at all. The PRESENCE of a secret is
// reported as a boolean/count; the value itself has no field here, so it cannot be read
// back out of this type even by accident.
type NetworkConfigPublic struct {
	NetworkName    string   `json:"network_name"`
	ExplorerURL    string   `json:"explorer_url"`
	IndexerURLs    []string `json:"indexer_urls"`
	NodeURLs       []string `json:"node_urls"`
	FaucetURL      string   `json:"faucet_url"`
	AssetID        string   `json:"asset_id"`
	AppID          string   `json:"app_id"`
	ChainID        string   `json:"chain_id"`
	PowerDivisor   float64  `json:"power_divisor"`
	PowerBase      int      `json:"power_base"`
	IPFSGatewayURL string   `json:"ipfs_gateway_url"`

	AlgodTokenConfigured bool `json:"algod_token_configured"`
	IPFSAPIKeyConfigured bool `json:"ipfs_api_key_configured"`
	IPFSHeaderCount      int  `json:"ipfs_header_count"`
}

// publicView projects one config. Secret values are DROPPED; their presence is REPORTED.
// URL lists are normalised to non-nil so the JSON carries [] and never null (an absent
// list is an empty list, the same contract the portfolio's association export uses).
func (c NetworkConfig) publicView() NetworkConfigPublic {
	indexers := make([]string, 0, len(c.IndexerURLs))
	indexers = append(indexers, c.IndexerURLs...)
	nodes := make([]string, 0, len(c.NodeURLs))
	nodes = append(nodes, c.NodeURLs...)

	return NetworkConfigPublic{
		NetworkName:    c.NetworkName,
		ExplorerURL:    c.ExplorerURL,
		IndexerURLs:    indexers,
		NodeURLs:       nodes,
		FaucetURL:      c.FaucetURL,
		AssetID:        c.AssetID,
		AppID:          c.AppID,
		ChainID:        c.ChainID,
		PowerDivisor:   c.PowerDivisor,
		PowerBase:      c.PowerBase,
		IPFSGatewayURL: c.IPFSGatewayURL,

		AlgodTokenConfigured: c.AlgodToken != "",
		IPFSAPIKeyConfigured: c.IPFSAPIKey != "",
		IPFSHeaderCount:      len(c.IPFSHeaders),
	}
}

// publicNetworkViews projects the whole registry for SERVING (broadcast + admin read).
func publicNetworkViews(in map[string]NetworkConfig) map[string]NetworkConfigPublic {
	out := make(map[string]NetworkConfigPublic, len(in))
	for name, cfg := range in {
		out[name] = cfg.publicView()
	}
	return out
}

// publicNetworkConfigsForDisk is the projection PERSISTED to networks.json. It stays a
// NetworkConfig map so the loader (which unmarshals into map[string]NetworkConfig) reads
// back exactly what was written — but with every secret cleared.
func publicNetworkConfigsForDisk(in map[string]NetworkConfig) map[string]NetworkConfig {
	out := make(map[string]NetworkConfig, len(in))
	for name, cfg := range in {
		cfg.AlgodToken = ""
		cfg.IPFSAPIKey = ""
		cfg.IPFSHeaders = nil
		out[name] = cfg
	}
	return out
}

// withheldSecretSummary counts the secret material the disk writer deliberately does NOT
// persist, naming the networks involved, so the omission is REPORTED and never silent.
func withheldSecretSummary(in map[string]NetworkConfig) (tokens, ipfsKeys, headerSets int, names []string) {
	for name, cfg := range in {
		found := false
		if cfg.AlgodToken != "" {
			tokens++
			found = true
		}
		if cfg.IPFSAPIKey != "" {
			ipfsKeys++
			found = true
		}
		if len(cfg.IPFSHeaders) > 0 {
			headerSets++
			found = true
		}
		if found {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return tokens, ipfsKeys, headerSets, names
}

// warnWithheldSecrets states that the registry file was persisted WITHOUT secret material.
// Silence here would read as "the registry holds no secrets", which is how an operator comes
// to hand-edit one back into a git-tracked file.
func warnWithheldSecrets(tokens, ipfsKeys, headerSets int, names []string) {
	if tokens == 0 && ipfsKeys == 0 && headerSets == 0 {
		return
	}
	log.Printf("[CONFIG] networks.json persisted WITHOUT secret material (%d algod token(s), %d IPFS api key(s), %d IPFS header set(s)) from network(s) %v. "+
		"Secrets are never written to the git-tracked registry; supply them via environment at boot (ALGOD_TOKEN_VOI / ALGOD_TOKEN_ALGO / IPFS_API_KEY).\n",
		tokens, ipfsKeys, headerSets, names)
}

// registrySecretKeyMarkers are the key-name fragments that would mean a secret VALUE is
// being served. They are checked against the SERVED KEYS, never against values.
var registrySecretKeyMarkers = []string{"token", "secret", "password", "mnemonic", "private", "api_key", "apikey", "authorization"}

// assertNetworkRegistryRedaction panics at startup if the public projection ever serves a
// secret-shaped KEY. The two presence flags (`..._configured`) and the header COUNT are
// exempt: they report presence, never value. This is the same fail-loud discipline the
// bonded-branding card rule uses, applied to the registry's public surface.
func assertNetworkRegistryRedaction() {
	probe, err := json.Marshal(NetworkConfigPublic{})
	if err != nil {
		panic("[CONFIG POLICY] NetworkConfigPublic cannot be serialised, so its redaction cannot be proven: " + err.Error())
	}
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(probe, &keys); err != nil {
		panic("[CONFIG POLICY] NetworkConfigPublic serialised into a shape that is not a JSON object: " + err.Error())
	}
	for key := range keys {
		if marker := secretShapedKey(key); marker != "" {
			panic("[CONFIG POLICY] NetworkConfigPublic serves the secret-shaped key \"" + key + "\" (marker \"" + marker +
				"\"). The network registry is broadcast to every client and persisted to a git-tracked file, so a secret value " +
				"must never be reachable through this projection. Report presence with a _configured flag instead.")
		}
	}
}

// secretShapedKey reports the marker that makes a SERVED key secret-shaped, or "" when the key
// is safe. Presence flags (`..._configured`) and counts (`..._count`) are exempt: they are the
// sanctioned way to report a secret's presence and carry no value.
func secretShapedKey(key string) string {
	lower := strings.ToLower(key)
	if strings.HasSuffix(lower, "_configured") || strings.HasSuffix(lower, "_count") {
		return ""
	}
	for _, marker := range registrySecretKeyMarkers {
		if strings.Contains(lower, marker) {
			return marker
		}
	}
	return ""
}

func init() {
	assertNetworkRegistryRedaction()
}

// envVarsDeclaredButUnread names the environment variables that .env.example documents but
// which NO Go code reads (verified by repo-wide grep when this list was written). Serving the
// list lets the admin console state the difference between "authoritative" and "advertised"
// instead of implying that editing one of these changes behaviour.
var envVarsDeclaredButUnread = []string{
	"ADMIN_KEY",               // the legacy X-Admin-Key fallback was removed from checkAdminAuth
	"MAX_FAUCET_CAPACITY",     // the capacity is the hard-coded default in newLobby()
	"COLLECTION_ID",           // no reader: the collection id is not used by any handler
	"FALLBACK_REWARD_ASSET_ID",// no reader: the fallback asset id is not used by any handler
	"VOI_GAS_THRESHOLD",       // no reader: gas warnings use their own literals
	"MIN_REPUTATION_FOR_REWARD", // no reader: the loyalty bonus threshold is not read from env
}

// handleAdminNetworks serves the READ-ONLY authority view of the network registry: the
// redacted configs, which networks hold secret material in the RUNNING process, and which env
// vars are actually authoritative. GET /api/admin/networks (registered in both servers).
func (l *Lobby) handleAdminNetworks(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed: the network registry authority view is read-only", http.StatusMethodNotAllowed)
		return
	}

	l.mutex.RLock()
	networks := publicNetworkViews(l.availableNetworks)
	tokens, ipfsKeys, headerSets, secretNetworks := withheldSecretSummary(l.availableNetworks)
	vaultAddress := l.vaultAddress
	rewardAssetID := l.rewardAssetID
	avoiAssetID := l.avoiAssetID
	baseRewardMicro := l.baseReward
	dataDir := l.DataDir
	l.mutex.RUnlock()

	adminWalletCount := 0
	for _, candidate := range strings.Split(os.Getenv("ADMIN_WALLETS"), ",") {
		if strings.TrimSpace(candidate) != "" {
			adminWalletCount++
		}
	}
	mnemonicWords := len(strings.Fields(os.Getenv("FAUCET_MNEMONIC")))

	payload := map[string]interface{}{
		"success":  true,
		"networks": networks,
		"secrets_configured": map[string]interface{}{
			"networks_holding_secrets":    secretNetworks,
			"algod_tokens_configured":     tokens,
			"ipfs_api_keys_configured":    ipfsKeys,
			"ipfs_header_sets_configured": headerSets,
			"rule": "secret VALUES are never served and never persisted; only their presence is reported",
		},
		"env_authority": map[string]interface{}{
			"vault_address":               vaultAddress,
			"reward_asset_id":             rewardAssetID,
			"avoi_asset_id":               avoiAssetID,
			"base_reward_micro":           baseRewardMicro,
			"faucet_mnemonic_configured":  mnemonicWords > 0,
			"faucet_mnemonic_word_count":  mnemonicWords,
			"admin_wallets_configured":    adminWalletCount,
			"algod_token_voi_configured":  os.Getenv("ALGOD_TOKEN_VOI") != "",
			"algod_token_algo_configured": os.Getenv("ALGOD_TOKEN_ALGO") != "",
			"ipfs_api_key_configured":     os.Getenv("IPFS_API_KEY") != "",
			"default_network":             os.Getenv("DEFAULT_NETWORK"),
			"data_dir":                    dataDir,
			"wc_project_id_configured":    os.Getenv("WC_PROJECT_ID") != "" || os.Getenv("AppID") != "",
			"rule":                        "these values are ENV-owned: this surface READS them and can never write them",
		},
		"declared_but_unread": envVarsDeclaredButUnread,
		"registry_file":       "networks.json",
		"registry_writable":   true,
		"restart_required":    true,
		"note": "A registry change (POST /api/admin/network/add) is persisted immediately, but the map is loaded at BOOT, so the server must be restarted before readers use a new indexer/node base.",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("[CONFIG ERROR] unable to serve the network registry authority view: %v\n", err)
	}
}
