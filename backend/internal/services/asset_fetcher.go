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
type RawAsset struct {
	TypeID       int
	LocationID   int64
	Quantity     int
	LocationFlag string
}

// assetFetcher fetches a character's full asset list (keeps AssetSaleService testable).
type assetFetcher interface {
	FetchCharacterAssets(ctx context.Context, characterID int, accessToken string) ([]RawAsset, error)
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
	TypeID       int    `json:"type_id"`
	LocationID   int64  `json:"location_id"`
	Quantity     int    `json:"quantity"`
	LocationFlag string `json:"location_flag"`
}

// FetchCharacterAssets pulls every page of the character's assets.
func (f *ESIAssetFetcher) FetchCharacterAssets(ctx context.Context, characterID int, accessToken string) ([]RawAsset, error) {
	base := fmt.Sprintf("%s/latest/characters/%d/assets/", f.baseURL, characterID)
	first, pages, err := f.page(ctx, base, accessToken, 1)
	if err != nil {
		return nil, err
	}
	all := first
	for p := 2; p <= pages; p++ {
		more, _, err := f.page(ctx, base, accessToken, p)
		if err != nil {
			return nil, err
		}
		all = append(all, more...)
	}
	return all, nil
}

func (f *ESIAssetFetcher) page(ctx context.Context, base, token string, page int) ([]RawAsset, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"?page="+strconv.Itoa(page), nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, 0, fmt.Errorf("unauthorized")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("ESI assets status %d", resp.StatusCode)
	}
	pages := 1
	if x := resp.Header.Get("X-Pages"); x != "" {
		if v, err := strconv.Atoi(x); err == nil {
			pages = v
		}
	}
	var raw []esiAssetPage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, 0, err
	}
	out := make([]RawAsset, 0, len(raw))
	for _, a := range raw {
		out = append(out, RawAsset{TypeID: a.TypeID, LocationID: a.LocationID, Quantity: a.Quantity, LocationFlag: a.LocationFlag})
	}
	return out, pages, nil
}
