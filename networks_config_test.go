package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// TestNetworkRegistryLoadsEveryChain guards the shape of the on-disk network registry.
//
// Why this exists: `networks.json` is the AUTHORITATIVE runtime config
// (Lobby.loadNetworkConfigs reads it and falls back to hardcoded defaults only when the
// file is missing), and its keys must match the NetworkConfig json tags exactly. Six of the
// eight shipped entries previously stored "node_url" (SINGULAR) while the struct tag is
// "node_urls" ([]string), so those chains unmarshalled to an EMPTY NodeURLs slice and every
// `cfg.NodeURLs[0]` consumer silently had no RPC to talk to. A wrong key does not raise an
// error — it just answers nothing — which is exactly the class of defect a test must pin.
func TestNetworkRegistryLoadsEveryChain(t *testing.T) {
	data, err := os.ReadFile("networks.json")
	if err != nil {
		t.Fatalf("networks.json must exist in the repository root: %v", err)
	}

	var registry map[string]NetworkConfig
	if err := json.Unmarshal(data, &registry); err != nil {
		t.Fatalf("networks.json must decode into the live NetworkConfig map: %v", err)
	}
	if len(registry) == 0 {
		t.Fatal("networks.json decoded to an empty registry")
	}

	for name, cfg := range registry {
		if len(cfg.IndexerURLs) == 0 {
			t.Errorf("%s: no indexer_urls — every chain read (ARC-200 transfers, checkpoints) goes through an indexer", name)
		}
		for _, u := range cfg.IndexerURLs {
			if u == "" {
				t.Errorf("%s: indexer_urls contains an empty entry", name)
			}
		}
		if len(cfg.NodeURLs) == 0 {
			t.Errorf("%s: no node_urls — the config key must be plural (\"node_url\"/singular decodes to nothing)", name)
		}
		if cfg.ChainID == "" {
			t.Errorf("%s: no chain_id", name)
		}
	}

	// The primary network the game settles on must be intact and pointed at the public
	// Voi indexer; a corrupted Voi entry breaks money-door verification.
	voi, ok := registry["Voi Mainnet"]
	if !ok {
		t.Fatal("Voi Mainnet is the primary settlement network and must be registered")
	}
	if len(voi.IndexerURLs) == 0 || voi.IndexerURLs[0] == "" {
		t.Errorf("Voi Mainnet: indexer_urls must be set (got %v)", voi.IndexerURLs)
	}
	if len(voi.NodeURLs) == 0 || voi.NodeURLs[0] == "" {
		t.Errorf("Voi Mainnet: node_urls must be set (got %v)", voi.NodeURLs)
	}
	if voi.ChainID == "" {
		t.Error("Voi Mainnet: chain_id must be set")
	}

	// ── TWO INDEXER FAMILIES ARE REQUIRED ON VOI (measured 2026-09-17) ──────
	//
	// The Voi chain exposes its ARC-200 API and its standard indexer API on
	// DIFFERENT hosts, and neither host serves the other's paths:
	//
	//   https://voi-mainnet-mimirapi.nftnavigator.xyz  -> /arc200/* + /nft-indexer/v1/*
	//                                                     (/v2/* answers 404)
	//   https://mainnet-idx.voi.nodely.dev             -> /v2/* (/arc200/* answers 404)
	//
	// indexerGet advances to the NEXT base on a 404, so listing both makes every
	// read land on the host that serves it — but only while BOTH are present. A
	// registry keeping just one of them silently 404s half the chain reads (the
	// money-door transfer lookup, the vault balance, the checkpoint reads), and a
	// wrong value raises no error at all.
	if !hasHost(voi.IndexerURLs, "voi-mainnet-mimirapi.nftnavigator.xyz") {
		t.Errorf("Voi Mainnet: indexer_urls must include the ARC-200 API host (got %v) — no other configured base serves /arc200/*", voi.IndexerURLs)
	}
	if !hasHost(voi.IndexerURLs, "mainnet-idx.voi.nodely.dev") {
		t.Errorf("Voi Mainnet: indexer_urls must include the standard indexer host (got %v) — no other configured base serves /v2/*, which the note-prefix checkpoint read requires", voi.IndexerURLs)
	}

	// The ARC-200 token identity is NAMED, not inferred: 40227315 is what the
	// chain reports for $VBV ("Virtual Babes VOiconomy", decimals 6) and the id
	// the vault's balance is read through. An empty asset_id leaves every money
	// path falling back — or resolving nothing.
	if !isDecimalID(voi.AssetID) {
		t.Errorf("Voi Mainnet: asset_id must be a non-empty decimal ARC-200 id (got %q)", voi.AssetID)
	}
	if !isDecimalID(voi.AppID) {
		t.Errorf("Voi Mainnet: app_id must be a non-empty decimal id (got %q) — server.go parses it into multiChainRouter.VoiAssetID", voi.AppID)
	}

	// Every network whose app id is set must be PARSEABLE as an uint64: server.go
	// does strconv.ParseUint(cfg.AppID, 10, 64) and a parse failure is silent —
	// the router's app id simply stays 0 and the app-call path cannot work.
	// A literal "0" is the legitimate spelling for "this chain has no game app"
	// (Algorand Mainnet ships it), so only a NON-numeric value is an error.
	for name, cfg := range registry {
		if cfg.AppID == "" || cfg.AppID == "0" {
			continue
		}
		if !isDecimalID(cfg.AppID) {
			t.Errorf("%s: app_id %q is not a decimal uint64 (server.go parses it and silently keeps 0 on failure)", name, cfg.AppID)
		}
	}
}

// hasHost reports whether any configured base URL contains host.
func hasHost(urls []string, host string) bool {
	for _, u := range urls {
		if strings.Contains(u, host) {
			return true
		}
	}
	return false
}

// isDecimalID reports whether s is a non-empty decimal number that is not zero
// (both "" and "0" are ids that can never resolve an asset).
func isDecimalID(s string) bool {
	if s == "" || s == "0" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
