package sm

import (
	"strings"
)

func extractSmContextRef(headers map[string]string) string {
	return extractIdFromLocation(headers)
}

func extractIdFromLocation(headers map[string]string) string {
	if v, ok := headers["Location"]; ok {
		if parts := strings.Split(v, "/"); len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}
	return ""
}
