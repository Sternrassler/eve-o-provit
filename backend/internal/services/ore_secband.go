package services

// Curated ore-group → security-band availability. The SDE does not cleanly encode
// where each ore spawns, so this is a maintained mapping of the classic asteroid
// ore groups. Group ids are from the SDE 'Asteroid' category. Special groups (Ice,
// moon/deadspace/AIR asteroids, decorative) are intentionally excluded.
//
// Tiers are cumulative: lowsec includes highsec ores; nullsec includes lowsec ores.
var (
	highSecOreGroups = []int64{
		462, // Veldspar
		460, // Scordite
		458, // Plagioclase
		459, // Pyroxeres
		469, // Omber
		457, // Kernite
		456, // Jaspet
		455, // Hemorphite
		454, // Hedbergite
	}
	lowSecExtraOreGroups = []int64{
		467, // Gneiss
		453, // Dark Ochre
		452, // Crokite
	}
	nullSecExtraOreGroups = []int64{
		450, // Arkonor
		451, // Bistot
		461, // Spodumain
		468, // Mercoxit
	}
)

// secBandOreGroups returns the set of ore group ids available in a security band.
// Unknown bands return an empty set.
func secBandOreGroups(band string) map[int64]bool {
	out := map[int64]bool{}
	add := func(ids []int64) {
		for _, id := range ids {
			out[id] = true
		}
	}
	switch band {
	case "null":
		add(nullSecExtraOreGroups)
		fallthrough
	case "low":
		add(lowSecExtraOreGroups)
		fallthrough
	case "high":
		add(highSecOreGroups)
	}
	return out
}
