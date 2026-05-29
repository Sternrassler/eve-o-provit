package services

import "github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/navigation"

// shipSpeed carries a ship's effective (skill- and fitting-derived) warp speed
// in AU/s and align time in seconds. Zero means "unknown".
type shipSpeed struct {
	warpAUS   float64
	alignTime float64
}

// shipNavParams builds NavigationParams from a ship's effective speed so the
// travel-time calc reflects the actual ship (a fast, agile ship does more
// trips per time budget). Zero values are left unset, so CalculateTravelTime
// falls back to its navigation defaults. Shared by hauling + sell-assets; the
// route calculator threads the same values through its own path.
func shipNavParams(s shipSpeed, avoidLowSec bool) *navigation.NavigationParams {
	p := &navigation.NavigationParams{AvoidLowSec: avoidLowSec}
	if s.warpAUS > 0 {
		w := s.warpAUS
		p.WarpSpeed = &w
	}
	if s.alignTime > 0 {
		a := s.alignTime
		p.AlignTime = &a
	}
	return p
}
