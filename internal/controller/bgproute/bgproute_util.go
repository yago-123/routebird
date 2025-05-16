package controller

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
)

// calculateCMapHash generates a deterministic hash based on the ConfigMap's data content.
func calculateCMapHash(data map[string]string) string {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	// Ensure consistent ordering
	sort.Strings(keys)

	hasher := sha256.New()
	for _, k := range keys {
		hasher.Write([]byte(k))
		hasher.Write([]byte(data[k]))
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

// withExtraLabels returns a new map containing all entries from base,
// plus all entries from extra (overriding base on key collisions).
func withExtraLabels(base, extra map[string]string) map[string]string {
	merged := make(map[string]string, len(base)+len(extra))
	// copy base
	for k, v := range base {
		merged[k] = v
	}
	// copy/override from extra
	for k, v := range extra {
		merged[k] = v
	}
	return merged
}
