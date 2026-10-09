//go:build !js && !wasm

package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/algorand/go-algorand-sdk/v2/crypto"
)

// ════════════════════════════════════════════════════════════════════════════
// SLICE 1 — ENV-ONLY SECRETS · THE REGISTRY'S PUBLIC VIEW (network_registry_view.go)
//
// The registry is broadcast to every client by `lobby_update` and persisted to
// networks.json, which is GIT-TRACKED. These tests pin the ONE projection both doors
// use, so a secret VALUE has no route to either the wire or the disk.
// ════════════════════════════════════════════════════════════════════════════

const (
	testAlgodSecret  = "SUPER-SECRET-ALGOD-TOKEN"
	testIPFSSecret   = "SUPER-SECRET-IPFS-KEY"
	testHeaderSecret = "SUPER-SECRET-HEADER-VALUE"
)

// secretBearingLobby is a registry holding every kind of secret the registry can carry.
// DataDir points at a per-test temp dir so the audit log the admin paths write stays out of
// the repository's dev data.
func secretBearingLobby(t *testing.T) *Lobby {
	t.Helper()
	return &Lobby{
		vaultAddress:   "VAULT-ADDR-TEST",
		rewardAssetID:  "40227315",
		avoiAssetID:    "2320775407",
		baseReward:     5_000_000,
		DataDir:        t.TempDir(),
		initialRewards: map[string]uint64{"40227315": 5_000_000},
		nonces:         map[string]NonceData{},
		availableNetworks: map[string]NetworkConfig{
			"Voi Mainnet": {
				NetworkName: "Voi Mainnet",
				IndexerURLs: []string{"https://mainnet-idx.voi.nodely.dev"},
				NodeURLs:    []string{"https://mainnet-api.voi.nodely.dev"},
				ChainID:     "algorand:r20fSQI8gWe_kFZziNonSPCXLwcQmH_n",
				AssetID:     "40227315",
				AlgodToken:  testAlgodSecret,
				IPFSAPIKey:  testIPFSSecret,
				IPFSHeaders: map[string]string{"X-Api-Secret": testHeaderSecret},
			},
			"Algorand Mainnet": {
				NetworkName: "Algorand Mainnet",
				ChainID:     "algorand:wGHE2Pwdvd7S12BL5FaOP20EGYesN73k",
			},
		},
	}
}

func TestNetworkRegistryBroadcastCarriesNoSecret(t *testing.T) {
	l := secretBearingLobby(t)
	views := publicNetworkViews(l.availableNetworks)

	blob, err := json.Marshal(views)
	if err != nil {
		t.Fatalf("marshal the public views: %v", err)
	}
	encoded := string(blob)

	// 1. No secret VALUE may reach the wire.
	for _, secret := range []string{testAlgodSecret, testIPFSSecret, testHeaderSecret} {
		if strings.Contains(encoded, secret) {
			t.Errorf("the client projection leaked the secret value %q", secret)
		}
	}
	// 2. No secret-shaped KEY may reach the wire either: a "token" field on the wire is what a
	//    later edit would fill back in.
	for _, key := range []string{"algod_token\"", "ipfs_api_key\"", "ipfs_headers"} {
		if strings.Contains(encoded, key) {
			t.Errorf("the client projection serves the secret key %q", key)
		}
	}
	// 3. The projection is still COMPLETE for the operator UI (this is the field set the
	//    client's fillAdminNetworkForm/onAdminNetworkSelectChange actually read).
	voi := views["Voi Mainnet"]
	if voi.IndexerURLs[0] != "https://mainnet-idx.voi.nodely.dev" || voi.NodeURLs[0] != "https://mainnet-api.voi.nodely.dev" {
		t.Errorf("the projection dropped the endpoints the client renders: %+v", voi)
	}
	if voi.AssetID != "40227315" || voi.ChainID == "" || voi.NetworkName != "Voi Mainnet" {
		t.Errorf("the projection dropped identity fields: %+v", voi)
	}
	// 4. Presence of the secrets IS reported — an operator can see they are configured without
	//    the value ever being readable.
	if !voi.AlgodTokenConfigured || !voi.IPFSAPIKeyConfigured || voi.IPFSHeaderCount != 1 {
		t.Errorf("secret PRESENCE must be reported: %+v", voi)
	}
	if views["Algorand Mainnet"].AlgodTokenConfigured {
		t.Error("a network with no token must report algod_token_configured=false")
	}
	// 5. An absent list is an EMPTY list, never null (the registry/portfolio JSON contract).
	algo := views["Algorand Mainnet"]
	if algo.IndexerURLs == nil || algo.NodeURLs == nil {
		t.Error("absent URL lists must serialise as [] and never null")
	}
}

func TestNetworkRegistryDiskWriteCarriesNoSecret(t *testing.T) {
	l := secretBearingLobby(t)
	persisted := publicNetworkConfigsForDisk(l.availableNetworks)

	blob, err := json.MarshalIndent(persisted, "", "  ")
	if err != nil {
		t.Fatalf("marshal the persisted registry: %v", err)
	}
	encoded := string(blob)
	for _, secret := range []string{testAlgodSecret, testIPFSSecret, testHeaderSecret} {
		if strings.Contains(encoded, secret) {
			t.Errorf("networks.json would have carried the secret value %q", secret)
		}
	}

	// The REGISTRY itself must survive redaction intact: this is the file the server reloads.
	voi := persisted["Voi Mainnet"]
	if voi.IndexerURLs[0] != "https://mainnet-idx.voi.nodely.dev" || voi.NodeURLs[0] != "https://mainnet-api.voi.nodely.dev" {
		t.Errorf("redaction dropped the endpoints the loader needs: %+v", voi)
	}
	if voi.AssetID != "40227315" || voi.ChainID == "" {
		t.Errorf("redaction dropped registry identity: %+v", voi)
	}
	if voi.AlgodToken != "" || voi.IPFSAPIKey != "" || voi.IPFSHeaders != nil {
		t.Errorf("redaction left secret material to be written to disk: %+v", voi)
	}

	// ...and the RUNNING config keeps its secrets: the server still has to authenticate.
	if l.availableNetworks["Voi Mainnet"].AlgodToken != testAlgodSecret {
		t.Error("the disk projection must not clear the live config's token")
	}
	if l.availableNetworks["Voi Mainnet"].IPFSHeaders["X-Api-Secret"] != testHeaderSecret {
		t.Error("the disk projection must not mutate the live config's header map")
	}
}

func TestWithheldSecretSummarySortsAndIsSilentWhenClean(t *testing.T) {
	l := secretBearingLobby(t)
	tokens, ipfsKeys, headerSets, names := withheldSecretSummary(l.availableNetworks)
	if tokens != 1 || ipfsKeys != 1 || headerSets != 1 {
		t.Errorf("summary = (%d,%d,%d); want (1,1,1)", tokens, ipfsKeys, headerSets)
	}
	if len(names) != 1 || names[0] != "Voi Mainnet" {
		t.Errorf("named networks = %v; want exactly [Voi Mainnet] in deterministic order", names)
	}

	// A registry holding no secret reports none: "nothing withheld" must be the only reading.
	clean := map[string]NetworkConfig{"Ethereum": {NetworkName: "Ethereum"}}
	if tk, ik, hs, nm := withheldSecretSummary(clean); tk != 0 || ik != 0 || hs != 0 || len(nm) != 0 {
		t.Errorf("a clean registry reported withheld secrets: (%d,%d,%d,%v)", tk, ik, hs, nm)
	}

	// Sorting is by network name, not by map iteration order.
	multi := map[string]NetworkConfig{
		"WAX":         {AlgodToken: "a"},
		"Bitcoin":     {IPFSAPIKey: "b"},
		"Voi Mainnet": {IPFSHeaders: map[string]string{"k": "v"}},
	}
	if _, _, _, n := withheldSecretSummary(multi); strings.Join(n, ",") != "Bitcoin,Voi Mainnet,WAX" {
		t.Errorf("withheld network names = %v; want [Bitcoin Voi Mainnet WAX]", n)
	}
}

func TestSecretShapedKeyRule(t *testing.T) {
	// Served keys that WOULD be a leak, with the marker that catches each.
	for key, wantMarker := range map[string]string{
		"algod_token":     "token",
		"ipfs_api_key":    "api_key",
		"apiKey":          "apikey",
		"client_secret":   "secret",
		"faucet_mnemonic": "mnemonic",
		"private_key":     "private",
		"authorization":   "authorization",
		"db_password":     "password",
	} {
		if got := secretShapedKey(key); got != wantMarker {
			t.Errorf("secretShapedKey(%q) = %q; want %q", key, got, wantMarker)
		}
	}

	// Presence flags, counts and every PUBLIC field of the projection are allowed.
	for _, key := range []string{
		"algod_token_configured", "ipfs_api_key_configured", "ipfs_header_count",
		"network_name", "explorer_url", "indexer_urls", "node_urls", "faucet_url",
		"asset_id", "app_id", "chain_id", "power_divisor", "power_base", "ipfs_gateway_url",
	} {
		if got := secretShapedKey(key); got != "" {
			t.Errorf("secretShapedKey(%q) = %q; a presence flag, a count or a public field must be allowed", key, got)
		}
	}
}

// adminTestAuth plants a REAL admin identity: a generated wallet in ADMIN_WALLETS, a live nonce
// and a genuine ARC-14 signature over the exact message the server verifies. The admin gate is
// therefore exercised for real, never bypassed by the test.
func adminTestAuth(t *testing.T, l *Lobby) map[string]string {
	t.Helper()

	// The SDK's GenerateAccount returns a single Account value (no error).
	acct := crypto.GenerateAccount()
	wallet := acct.Address.String()
	t.Setenv("ADMIN_WALLETS", wallet)

	nonce := fmt.Sprintf("slice1-nonce-%d", time.Now().UnixNano())
	if l.nonces == nil {
		l.nonces = map[string]NonceData{}
	}
	l.nonces[wallet] = NonceData{Value: nonce, CreatedAt: time.Now()}

	msg := fmt.Sprintf("Algorand Signed Message:\nVirtualbabes Arena Admin Auth:%s", nonce)
	// SDK v2 order: SignBytes(privateKey, bytesToSign). VerifyBytes checks the same message
	// against the raw 32 bytes of the address (which ARE the ed25519 public key), so this test
	// exercises the server's real gate.
	sig, err := crypto.SignBytes(acct.PrivateKey, []byte(msg))
	if err != nil {
		t.Fatalf("sign the admin nonce: %v", err)
	}

	return map[string]string{
		"X-Admin-Wallet":    wallet,
		"X-Admin-Nonce":     nonce,
		"X-Admin-Signature": base64.StdEncoding.EncodeToString(sig),
	}
}

func adminRequest(method, target string, headers map[string]string, body string) (*httptest.ResponseRecorder, *http.Request) {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return httptest.NewRecorder(), req
}

// adminAuthorityPayload is the served shape of GET /api/admin/networks.
type adminAuthorityPayload struct {
	Networks          map[string]NetworkConfigPublic `json:"networks"`
	SecretsConfigured struct {
		NetworksHoldingSecrets []string `json:"networks_holding_secrets"`
		AlgodTokens            int      `json:"algod_tokens_configured"`
		IPFSKeys               int      `json:"ipfs_api_keys_configured"`
		IPFSHeaderSets         int      `json:"ipfs_header_sets_configured"`
		Rule                   string   `json:"rule"`
	} `json:"secrets_configured"`
	EnvAuthority struct {
		VaultAddress             string `json:"vault_address"`
		RewardAssetID            string `json:"reward_asset_id"`
		BaseRewardMicro          uint64 `json:"base_reward_micro"`
		FaucetMnemonicConfigured bool   `json:"faucet_mnemonic_configured"`
		FaucetMnemonicWordCount  int    `json:"faucet_mnemonic_word_count"`
		AdminWalletsConfigured   int    `json:"admin_wallets_configured"`
		Rule                     string `json:"rule"`
	} `json:"env_authority"`
	DeclaredButUnread []string `json:"declared_but_unread"`
	RegistryFile      string   `json:"registry_file"`
	RestartRequired   bool     `json:"restart_required"`
}

func containsString(haystack []string, needle string) bool {
	for _, candidate := range haystack {
		if candidate == needle {
			return true
		}
	}
	return false
}

// TestAdminNetworksIsGatedReadOnlyAndLeakFree pins the whole contract of the authority view:
// refused without an admin signature, redacted with one, read-only, and honest about which env
// vars are actually authoritative.
func TestAdminNetworksIsGatedReadOnlyAndLeakFree(t *testing.T) {
	const testMnemonic = "alpha bravo charlie delta echo"
	l := secretBearingLobby(t)
	t.Setenv("FAUCET_MNEMONIC", testMnemonic)

	// 1. With no admin identity the view is refused outright.
	anonRec, anonReq := adminRequest(http.MethodGet, "/api/admin/networks", nil, "")
	l.handleAdminNetworks(anonRec, anonReq)
	if anonRec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous GET = %d; want 401", anonRec.Code)
	}

	headers := adminTestAuth(t, l)

	// 2. An authorised GET serves the view, and it still carries no secret.
	rec, req := adminRequest(http.MethodGet, "/api/admin/networks", headers, "")
	l.handleAdminNetworks(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("authorised GET = %d; want 200 (%s)", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, secret := range append([]string{testAlgodSecret, testIPFSSecret, testHeaderSecret}, testMnemonic) {
		if strings.Contains(body, secret) {
			t.Errorf("the authority view leaked %q", secret)
		}
	}

	var payload adminAuthorityPayload
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode the authority view: %v", err)
	}

	if !payload.Networks["Voi Mainnet"].AlgodTokenConfigured {
		t.Error("the view must report that a token IS configured (presence, never value)")
	}
	if payload.SecretsConfigured.AlgodTokens != 1 || payload.SecretsConfigured.IPFSKeys != 1 || payload.SecretsConfigured.IPFSHeaderSets != 1 {
		t.Errorf("secrets census = %+v; want one of each", payload.SecretsConfigured)
	}
	if strings.Join(payload.SecretsConfigured.NetworksHoldingSecrets, ",") != "Voi Mainnet" {
		t.Errorf("networks holding secrets = %v; want [Voi Mainnet]", payload.SecretsConfigured.NetworksHoldingSecrets)
	}
	if payload.SecretsConfigured.Rule == "" {
		t.Error("the secrets block must state its own rule")
	}

	// The env authority block READS the environment: presence is reported, the value never is.
	if payload.EnvAuthority.VaultAddress != "VAULT-ADDR-TEST" || payload.EnvAuthority.RewardAssetID != "40227315" {
		t.Errorf("env authority = %+v; want the env-owned values", payload.EnvAuthority)
	}
	if payload.EnvAuthority.BaseRewardMicro != 5_000_000 {
		t.Errorf("base_reward_micro = %d; want 5000000 (micro units, never a float)", payload.EnvAuthority.BaseRewardMicro)
	}
	if !payload.EnvAuthority.FaucetMnemonicConfigured || payload.EnvAuthority.FaucetMnemonicWordCount != 5 {
		t.Errorf("mnemonic presence = %+v; want configured=true, 5 words", payload.EnvAuthority)
	}
	if payload.EnvAuthority.AdminWalletsConfigured != 1 {
		t.Errorf("admin_wallets_configured = %d; want 1", payload.EnvAuthority.AdminWalletsConfigured)
	}
	if payload.EnvAuthority.Rule == "" {
		t.Error("the env authority block must state that the panel can never write these")
	}

	if !containsString(payload.DeclaredButUnread, "ADMIN_KEY") || !containsString(payload.DeclaredButUnread, "MAX_FAUCET_CAPACITY") {
		t.Errorf("declared_but_unread = %v; want it to name ADMIN_KEY and MAX_FAUCET_CAPACITY", payload.DeclaredButUnread)
	}
	if payload.RegistryFile != "networks.json" || !payload.RestartRequired {
		t.Errorf("registry_file=%q restart_required=%v; want networks.json + true", payload.RegistryFile, payload.RestartRequired)
	}

	// 3. READ-ONLY: a write verb is refused with a reason.
	postRec, postReq := adminRequest(http.MethodPost, "/api/admin/networks", headers, `{"network_name":"X"}`)
	l.handleAdminNetworks(postRec, postReq)
	if postRec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST = %d; want 405 (registry writes belong to POST /api/admin/network/add)", postRec.Code)
	}
}

// TestUpdateRewardAssetIsRefusedAndWritesNothing pins the ENV authority of the primary reward
// asset: the door refuses, names the env var, and moves no state at all — in particular it does
// not leave a stale asset id in the reward template, which is what made BOTH tokens payable.
func TestUpdateRewardAssetIsRefusedAndWritesNothing(t *testing.T) {
	l := secretBearingLobby(t)
	beforeTemplate := len(l.initialRewards)
	headers := adminTestAuth(t, l)

	rec, req := adminRequest(http.MethodPost, "/api/reward/update-asset", headers, `{"asset_id":"999999"}`)
	l.handleUpdateRewardAsset(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("update-asset = %d; want 409 with a stated reason (%s)", rec.Code, rec.Body.String())
	}
	responseBody := rec.Body.String()
	if !strings.Contains(responseBody, "REWARD_ASSET_ID") {
		t.Error("the refusal must name the environment variable that owns the value")
	}
	if !strings.Contains(responseBody, "40227315") {
		t.Error("the refusal must state the CURRENT value so the operator sees what is in force")
	}

	l.mutex.RLock()
	defer l.mutex.RUnlock()
	if l.rewardAssetID != "40227315" {
		t.Errorf("reward_asset_id became %q; it is env-owned and must not be runtime-writable", l.rewardAssetID)
	}
	if _, stale := l.initialRewards["999999"]; stale {
		t.Error("a refused change must not enter the reward template")
	}
	if len(l.initialRewards) != beforeTemplate {
		t.Errorf("the refused change altered the template: %d keys, want %d", len(l.initialRewards), beforeTemplate)
	}
}
