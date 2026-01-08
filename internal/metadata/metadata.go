package metadata

import (
	"strconv"
	"strings"
)

var (
	Version    = "freshest"
	CommitHash = "n/a"
	BuildTime  = "n/a"
)

func IsMajorDifference(v1, v2 string) bool {
	if v1 == "freshest" || v2 == "freshest" {
		return false
	}

	major1 := extractMajor(v1)
	major2 := extractMajor(v2)

	return major1 != major2
}

func extractMajor(v string) int {
	v = strings.TrimPrefix(v, "v.")
	v = strings.TrimPrefix(v, "v")

	parts := strings.Split(v, ".")
	if len(parts) == 0 {
		return 0
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0
	}

	return major
}
