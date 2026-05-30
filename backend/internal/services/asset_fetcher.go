package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// RawAsset is the minimal ESI asset shape the aggregator needs.
//
// ItemID is the ESI item_id of *this* asset stack. We need it so that we can
// resolve "item-in-container" cases: a child asset's LocationID points to its
// container's ItemID (not to the station). Walking the chain bottom-up via
// ItemID → LocationID lookups eventually hits an SDE-known station.
type RawAsset struct {
	ItemID       int64
	TypeID       int
	LocationID   int64
	Quantity     int
	LocationFlag string
}

// assetFetcher fetches a character's full asset list (keeps AssetSaleService testable).
//
// expiresAt is the ESI cache-expiry time parsed from the `Expires` response
// header. Until that point, ESI serves the same snapshot — a client refresh
// won't see fresh data. Returns the zero value if the header was missing or
// unparseable; callers must treat that as "expiry unknown".
type assetFetcher interface {
	FetchCharacterAssets(ctx context.Context, characterID int, accessToken string) (assets []RawAsset, expiresAt time.Time, err error)
}

// ESIAssetFetcher calls /characters/{id}/assets/ with pagination (X-Pages).
type ESIAssetFetcher struct {
	baseURL string
	client  *http.Client
}

// NewESIAssetFetcher creates a fetcher against the live ESI base URL.
func NewESIAssetFetcher() *ESIAssetFetcher {
	return &ESIAssetFetcher{
		baseURL: "https://esi.evetech.net",
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

type esiAssetPage struct {
	ItemID       int64  `json:"item_id"`
	TypeID       int    `json:"type_id"`
	LocationID   int64  `json:"location_id"`
	Quantity     int    `json:"quantity"`
	LocationFlag string `json:"location_flag"`
}

// FetchCharacterAssets pulls every page of the character's assets and returns
// the ESI cache-expiry from the first page's `Expires` header.
func (f *ESIAssetFetcher) FetchCharacterAssets(ctx context.Context, characterID int, accessToken string) ([]RawAsset, time.Time, error) {
	base := fmt.Sprintf("%s/latest/characters/%d/assets/", f.baseURL, characterID)
	first, pages, expiresAt, err := f.page(ctx, base, accessToken, 1)
	if err != nil {
		return nil, time.Time{}, err
	}
	all := first
	for p := 2; p <= pages; p++ {
		more, _, _, err := f.page(ctx, base, accessToken, p)
		if err != nil {
			return nil, time.Time{}, err
		}
		all = append(all, more...)
	}
	return all, expiresAt, nil
}

func (f *ESIAssetFetcher) page(ctx context.Context, base, token string, page int) ([]RawAsset, int, time.Time, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"?page="+strconv.Itoa(page), nil)
	if err != nil {
		return nil, 0, time.Time{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, 0, time.Time{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, 0, time.Time{}, fmt.Errorf("unauthorized")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, 0, time.Time{}, fmt.Errorf("ESI assets status %d", resp.StatusCode)
	}
	pages := 1
	if x := resp.Header.Get("X-Pages"); x != "" {
		if v, err := strconv.Atoi(x); err == nil {
			pages = v
		}
	}
	// ESI's `Expires` header is RFC 1123 / http.TimeFormat. Best-effort: parse
	// it; on failure leave expiresAt as zero so the caller treats it as
	// "unknown" rather than crashing.
	var expiresAt time.Time
	if x := resp.Header.Get("Expires"); x != "" {
		if t, err := http.ParseTime(x); err == nil {
			expiresAt = t.UTC()
		}
	}
	var raw []esiAssetPage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, 0, time.Time{}, err
	}
	out := make([]RawAsset, 0, len(raw))
	for _, a := range raw {
		out = append(out, RawAsset{
			ItemID:       a.ItemID,
			TypeID:       a.TypeID,
			LocationID:   a.LocationID,
			Quantity:     a.Quantity,
			LocationFlag: a.LocationFlag,
		})
	}
	return out, pages, expiresAt, nil
}
