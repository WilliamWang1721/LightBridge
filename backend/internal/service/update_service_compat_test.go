//go:build unit

package service

func parseVersion(version string) [3]int {
	parts, _ := parseSemanticVersion(version)
	return parts
}
