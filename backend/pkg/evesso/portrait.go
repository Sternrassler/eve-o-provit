package evesso

import "fmt"

// CharacterInfo represents verified character information extracted from an EVE SSO JWT.
type CharacterInfo struct {
	CharacterID        int
	CharacterName      string
	Scopes             string // space-joined scope list (matches prior /verify/ format)
	CharacterOwnerHash string
}

// GetPortraitURL returns the character portrait URL.
func GetPortraitURL(characterID int, size int) string {
	return fmt.Sprintf("https://images.evetech.net/characters/%d/portrait?size=%d", characterID, size)
}
