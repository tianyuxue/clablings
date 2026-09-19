package content

import (
	"crypto/sha1"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
)

// InstanceID identifies one deployment instance of a library: CLABLINGS_INSTANCE
// when set, otherwise the first 8 hex chars of sha1(abs(root)).
func InstanceID(root string) string {
	if v := strings.TrimSpace(os.Getenv("CLABLINGS_INSTANCE")); v != "" {
		if id := sanitizeName(v); id != "" {
			return id
		}
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		abs = root
	}
	sum := sha1.Sum([]byte(abs))
	return hex.EncodeToString(sum[:])[:8]
}

// LabName derives the containerlab lab name for a chapter topology as
// <sanitized root basename>-<instance>-<sanitized topology name>.
func LabName(root, instance, topologyName string) string {
	base := sanitizeName(filepath.Base(filepath.Clean(root)))
	if base == "" {
		base = "clablings"
	}
	id := sanitizeName(instance)
	if id == "" {
		id = "0"
	}
	return base + "-" + id + "-" + sanitizeName(topologyName)
}

// sanitizeName lowercases and keeps [a-z0-9], collapsing every other run of
// characters into a single '-' and trimming leading/trailing '-'.
func sanitizeName(s string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}
