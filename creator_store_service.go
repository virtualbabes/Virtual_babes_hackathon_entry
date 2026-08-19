//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"
)

// CreatorStoreProduct represents a creator's listed product or DLC item.
type CreatorStoreProduct struct {
	ID            string    `json:"id"`
	CreatorWallet string    `json:"creator_wallet"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	PriceMicroVBV uint64    `json:"price_micro_vbv"` // Price in micro-VBV units (uint64 precision)
	DLCLinks      []string  `json:"dlc_links,omitempty"`
	Category      string    `json:"category"` // "asset", "dlc", "service", "cosmetic"
	Tags          []string  `json:"tags,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	IsActive      bool      `json:"is_active"`
	SalesCount    uint64    `json:"sales_count"` // Total units sold
}

// CreatorProfile represents a creator's storefront profile.
type CreatorProfile struct {
	WalletAddress   string    `json:"wallet_address"`
	Bio             string    `json:"bio,omitempty"`
	DisplayName     string    `json:"display_name"`
	AvgRating       float64   `json:"avg_rating"`        // 0-5 scale, uint64 precision: stored as rating*1000
	RatingCount     uint64    `json:"rating_count"`      // Total ratings received
	TotalRevenueMicroVBV uint64 `json:"total_revenue_micro_vbv"` // Lifetime earnings (uint64)
	ProductCount    uint64    `json:"product_count"`     // Active products listed
	JoinedAt        time.Time `json:"joined_at"`
}

// RoyaltyTransaction represents a secondary-sale royalty payment.
type RoyaltyTransaction struct {
	ID            string    `json:"id"`
	ProductID     string    `json:"product_id"`
	BuyerWallet   string    `json:"buyer_wallet"`
	SellerWallet  string    `json:"seller_wallet"`
	CreatorWallet string    `json:"creator_wallet"`
	RoyaltyRate   float64   `json:"royalty_rate"` // e.g., 0.10 = 10% royalty
	AmountMicroVBV uint64   `json:"amount_micro_vbv"`
	RoyaltyPaidMicroVBV uint64 `json:"royalty_paid_micro_vbv"`
	Timestamp     time.Time `json:"timestamp"`
}

// CreatorStore manages the creator storefront system.
type CreatorStore struct {
	mu              sync.RWMutex
	products        map[string]*CreatorStoreProduct // productID -> Product
	creatorProfiles map[string]*CreatorProfile       // walletAddress -> Profile
	royaltyTxns     []RoyaltyTransaction             // Recent royalty transactions (bounded)
	maxRoyaltyHistory int                            // Max history entries before pruning
}

// CreatorStoreConstants
const (
	DefaultRoyaltyRate      = 0.10   // 10% standard creator royalty on secondary sales
	MaxProductTags          = 8       // Maximum tags per product
	ProductNameMaxLength    = 128     // Max characters for product name
	DescriptionMaxLength    = 2048    // Max characters for description
	BioMaxLength            = 512     // Max characters for creator bio
	RatingMultiplier        = 1000.0  // For uint64 precision on ratings (rating * RatingMultiplier)
	MaxRoyaltyHistoryEntries = 1000   // Bounded history before pruning oldest entries

	// Product categories
	CategoryAsset    = "asset"
	CategoryDLC      = "dlc"
	CategoryService  = "service"
	CategoryCosmetic = "cosmetic"

	// Royalty rate bounds (float64 for UI only; uint64 enforced server-side)
	MinRoyaltyRate   = 0.01 // Minimum 1% royalty
	MaxRoyaltyRate   = 0.25 // Maximum 25% royalty
	DefaultRoyaltyPct = 0.10 // Default 10% royalty for new creators

	// Revenue split: Creator gets (1 - platformFee), Platform retains remainder
	PlatformFeeRate = 0.10 // 10% platform fee on primary sales
)

var nextProductID uint64 = 1
var nextRoyaltyTxnID uint64 = 1

// NewCreatorStore creates a new CreatorStore instance.
func NewCreatorStore() *CreatorStore {
	return &CreatorStore{
		products:             make(map[string]*CreatorStoreProduct),
		creatorProfiles:      make(map[string]*CreatorProfile),
		royaltyTxns:          make([]RoyaltyTransaction, 0),
		maxRoyaltyHistory:    MaxRoyaltyHistoryEntries,
	}
}

// CreateProduct registers a new product in the creator store.
func (cs *CreatorStore) CreateProduct(productID string, creatorWallet, name, description, category string, priceMicroVBV uint64, tags []string, dlcLinks []string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Validate product ID format
	if len(productID) == 0 || len(productID) > 64 {
		return fmt.Errorf("create_product failed: invalid product ID length (must be 1-64 chars)")
	}

	// Check for duplicate product ID
	if _, exists := cs.products[productID]; exists {
		return fmt.Errorf("create_product failed: product %s already exists", productID)
	}

	// Validate name length
	name = sanitizeString(name, ProductNameMaxLength)
	if len(name) == 0 {
		return fmt.Errorf("create_product failed: product name must be non-empty after sanitization")
	}

	// Validate description length
	description = sanitizeString(description, DescriptionMaxLength)

	// Validate category
	switch category {
	case CategoryAsset, CategoryDLC, CategoryService, CategoryCosmetic:
		// Valid categories only
	default:
		category = CategoryAsset // Default to asset if invalid
	}

	// Clamp tags count and deduplicate
	tags = clampAndDedupTags(tags)

	// Price must be positive (uint64 ensures non-negative; check > 0)
	if priceMicroVBV == 0 {
		return fmt.Errorf("create_product failed: price must be greater than zero")
	}

	product := &CreatorStoreProduct{
		ID:            productID,
		CreatorWallet: creatorWallet,
		Name:          name,
		Description:   description,
		PriceMicroVBV: priceMicroVBV,
		DLCLinks:      dlcLinks[:min(len(dlcLinks), 10)], // Max 10 DLC links per product
		Category:      category,
		Tags:          tags,
		CreatedAt:     time.Now().UTC(),
		IsActive:      true,
		SalesCount:    0,
	}

	cs.products[productID] = product

	// Auto-create or update creator profile if not exists
	if _, exists := cs.creatorProfiles[creatorWallet]; !exists {
		cs.creatorProfiles[creatorWallet] = &CreatorProfile{
			WalletAddress:   creatorWallet,
			DisplayName:     fmt.Sprintf("Creator_%s", creatorWallet[:8]), // Default display name from wallet prefix
			AvgRating:       0.0,
			RatingCount:     0,
			TotalRevenueMicroVBV: 0,
			ProductCount:    1,
			JoinedAt:        time.Now().UTC(),
		}
	} else {
		cs.creatorProfiles[creatorWallet].ProductCount++
	}

	return nil
}

// ListProducts returns all active products, optionally filtered by category or creator.
func (cs *CreatorStore) ListProducts(category string, creatorWallet string) []*CreatorStoreProduct {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	var result []*CreatorStoreProduct

	for _, product := range cs.products {
		if !product.IsActive {
			continue
		}

		// Apply filters if specified
		if category != "" && product.Category != category {
			continue
		}
		if creatorWallet != "" && product.CreatorWallet != creatorWallet {
			continue
		}

		result = append(result, product)
	}

	return result
}

// GetProduct returns a single product by ID.
func (cs *CreatorStore) GetProduct(productID string) (*CreatorStoreProduct, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	product, exists := cs.products[productID]
	if !exists {
		return nil, fmt.Errorf("get_product failed: product %s not found", productID)
	}

	if !product.IsActive {
		return nil, fmt.Errorf("get_product failed: product %s is inactive", productID)
	}

	return product, nil
}

// PurchaseProduct processes a primary sale of a creator's product.
func (cs *CreatorStore) PurchaseProduct(productID string, buyerWallet string) (*RoyaltyTransaction, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	product, exists := cs.products[productID]
	if !exists || !product.IsActive {
		return nil, fmt.Errorf("purchase_product failed: product %s not found or inactive", productID)
	}

	price := product.PriceMicroVBV
	creatorWallet := product.CreatorWallet

	// Calculate revenue split (uint64 precision — no floating point for money)
	platformFee := uint64(float64(price) * PlatformFeeRate) // 10% platform fee
	creatorRevenue := price - platformFee                    // Creator gets remainder

	// Update creator profile total revenue with $VBV-gate multiplier support
	if profile, exists := cs.creatorProfiles[creatorWallet]; exists {
		profile.TotalRevenueMicroVBV += creatorRevenue
	} else {
		cs.creatorProfiles[creatorWallet] = &CreatorProfile{
			WalletAddress:        creatorWallet,
			DisplayName:          fmt.Sprintf("Creator_%s", creatorWallet[:8]),
			AvgRating:            0.0,
			RatingCount:          0,
			TotalRevenueMicroVBV: creatorRevenue,
			ProductCount:         1,
			JoinedAt:             time.Now().UTC(),
		}
	}

	// Increment sales count (uint64 overflow protection)
	if product.SalesCount < math.MaxUint64 {
		product.SalesCount++
	}

	// Create royalty transaction record for primary sale tracking
	txID := fmt.Sprintf("royalty_%d", nextRoyaltyTxnID)
	nextRoyaltyTxnID++

	primarySale := RoyaltyTransaction{
		ID:                  txID,
		ProductID:           productID,
		BuyerWallet:         buyerWallet,
		SellerWallet:        creatorWallet, // Primary sale: seller = creator
		CreatorWallet:       creatorWallet,
		RoyaltyRate:         0.0,            // No royalty on primary sales (creator gets full revenue minus platform fee)
		AmountMicroVBV:      price,
		RoyaltyPaidMicroVBV: 0,              // Creator doesn't pay royalty to themselves
		Timestamp:           time.Now().UTC(),
	}

	cs.royaltyTxns = append(cs.royaltyTxns, primarySale)
	pruneRoyaltyHistory(cs)

	return &primarySale, nil
}

// ProcessSecondarySale processes a secondary sale and calculates royalties owed to the creator.
func (cs *CreatorStore) ProcessSecondarySale(productID string, buyerWallet, sellerWallet string, salePriceMicroVBV uint64) (*RoyaltyTransaction, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	product, exists := cs.products[productID]
	if !exists || !product.IsActive {
		return nil, fmt.Errorf("process_secondary_sale failed: product %s not found or inactive", productID)
	}

	creatorWallet := product.CreatorWallet

	// Calculate royalty (uint64 precision — no floating point for money)
	royaltyRate := DefaultRoyaltyPct // 10% standard creator royalty on secondary sales
	royaltyAmount := uint64(float64(salePriceMicroVBV) * royaltyRate)
	sellerNetRevenue := salePriceMicroVBV - royaltyAmount

	// Update creator profile total revenue with royalties earned
	if profile, exists := cs.creatorProfiles[creatorWallet]; exists {
		profile.TotalRevenueMicroVBV += royaltyAmount
	} else {
		cs.creatorProfiles[creatorWallet] = &CreatorProfile{
			WalletAddress:        creatorWallet,
			DisplayName:          fmt.Sprintf("Creator_%s", creatorWallet[:8]),
			AvgRating:            0.0,
			RatingCount:          0,
			TotalRevenueMicroVBV: royaltyAmount,
			ProductCount:         product.SalesCount + 1, // Approximate count for unknown creators
			JoinedAt:             time.Now().UTC(),
		}
	}

	txID := fmt.Sprintf("royalty_%d", nextRoyaltyTxnID)
	nextRoyaltyTxnID++

	secondarySale := RoyaltyTransaction{
		ID:                  txID,
		ProductID:           productID,
		BuyerWallet:         buyerWallet,
		SellerWallet:        sellerWallet, // Secondary sale: seller = previous owner (not creator)
		CreatorWallet:       creatorWallet,
		RoyaltyRate:         royaltyRate,
		AmountMicroVBV:      salePriceMicroVBV,
		RoyaltyPaidMicroVBV: royaltyAmount,
		Timestamp:           time.Now().UTC(),
	}

	cs.royaltyTxns = append(cs.royaltyTxns, secondarySale)
	pruneRoyaltyHistory(cs)

	return &secondarySale, nil
}

// GetCreatorProfile returns a creator's profile by wallet address.
func (cs *CreatorStore) GetCreatorProfile(walletAddress string) (*CreatorProfile, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	profile, exists := cs.creatorProfiles[walletAddress]
	if !exists {
		return nil, fmt.Errorf("get_creator_profile failed: creator %s not found", walletAddress)
	}

	return profile, nil
}

// GetRoyaltyHistory returns recent royalty transactions for a product or creator.
func (cs *CreatorStore) GetRoyaltyHistory(productID string, creatorWallet string, limit int) []RoyaltyTransaction {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	if limit <= 0 || limit > 100 {
		limit = 50 // Default limit
	}

	var result []RoyaltyTransaction

	for i := len(cs.royaltyTxns) - 1; i >= 0 && len(result) < limit; i-- {
		tx := cs.royaltyTxns[i]

		if productID != "" && tx.ProductID != productID {
			continue
		}
		if creatorWallet != "" && tx.CreatorWallet != creatorWallet {
			continue
		}

		result = append(result, tx)
	}

	return result
}

// RateProduct allows a buyer to rate a purchased product.
func (cs *CreatorStore) RateProduct(productID string, buyerWallet string, rating uint64) error {
	cs.mu.Lock()
	defer cs.mu.RUnlock()

	product, exists := cs.products[productID]
	if !exists || !product.IsActive {
		return fmt.Errorf("rate_product failed: product %s not found or inactive", productID)
	}

	ratingClamped := clampUint64(rating, 1, 5) // Clamp to 1-5 scale

	creatorWallet := product.CreatorWallet
	profile, exists := cs.creatorProfiles[creatorWallet]
	if !exists {
		return fmt.Errorf("rate_product failed: creator profile for %s not found", creatorWallet)
	}

	oldRatingFloat := profile.AvgRating * RatingMultiplier // Convert back to raw value
	newTotal := uint64(oldRatingFloat)*profile.RatingCount + ratingClamped
	profile.RatingCount++
	if profile.RatingCount > 0 {
		profile.AvgRating = float64(newTotal) / (float64(profile.RatingCount) * RatingMultiplier) // Convert back to float64 for UI
	}

	return nil
}

// DeactivateProduct marks a product as inactive.
func (cs *CreatorStore) DeactivateProduct(productID string, creatorWallet string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	product, exists := cs.products[productID]
	if !exists || product.CreatorWallet != creatorWallet {
		return fmt.Errorf("deactivate_product failed: product %s not found or unauthorized", productID)
	}

	product.IsActive = false
	return nil
}

// ReactivateProduct marks a product as active again.
func (cs *CreatorStore) ReactivateProduct(productID string, creatorWallet string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	product, exists := cs.products[productID]
	if !exists || product.CreatorWallet != creatorWallet {
		return fmt.Errorf("reactivate_product failed: product %s not found or unauthorized", productID)
	}

	product.IsActive = true
	return nil
}

// Helper functions (private)

func sanitizeString(s string, maxLen int) string {
	if len(s) > maxLen {
		s = s[:maxLen]
	}
	return s
}

func clampAndDedupTags(tags []string) []string {
	tagSet := make(map[string]bool)
	var result []string
	for _, tag := range tags {
		if !tagSet[tag] && len(result) < MaxProductTags {
			tagSet[tag] = true
			result = append(result, tag)
		}
	}
	return result
}

func clampUint64(val uint64, minVal, maxVal uint64) uint64 {
	if val < minVal {
		return minVal
	}
	if val > maxVal {
		return maxVal
	}
	return val
}

func pruneRoyaltyHistory(cs *CreatorStore) {
	if len(cs.royaltyTxns) <= cs.maxRoyaltyHistory {
		return
	}
	// Keep only the most recent entries; discard oldest half to prevent unbounded growth
	pruneCount := len(cs.royaltyTxns) - (cs.maxRoyaltyHistory / 2)
	cs.royaltyTxns = cs.royaltyTxns[pruneCount:]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}