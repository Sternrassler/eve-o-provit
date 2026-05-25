package evesso

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

const (
	VerifyURL = "https://esi.evetech.net/verify/"
)

// VerifyToken verifies the access token with EVE ESI and returns character information
func VerifyToken(ctx context.Context, accessToken string) (*CharacterInfo, error) {
	if accessToken == "" {
		return nil, errors.New("access token is empty")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", VerifyURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create verify request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to verify token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("verify request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var charInfo CharacterInfo
	if err := json.NewDecoder(resp.Body).Decode(&charInfo); err != nil {
		return nil, fmt.Errorf("failed to decode character info: %w", err)
	}

	return &charInfo, nil
}
