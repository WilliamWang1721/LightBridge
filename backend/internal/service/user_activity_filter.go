package service

import "strings"

const (
	// UserActivityFilterAny matches users with either API usage or a balance-change record.
	UserActivityFilterAny = "any"
	// UserActivityFilterUsage matches users with at least one usage log.
	UserActivityFilterUsage = "usage"
	// UserActivityFilterBalanceChange matches users with at least one balance-change record.
	UserActivityFilterBalanceChange = "balance_change"
	// UserActivityFilterNone matches users with neither API usage nor a balance-change record.
	UserActivityFilterNone = "none"
)

// NormalizeUserActivityFilter normalizes the public API value and accepts a few
// descriptive aliases for automation clients. The canonical values are any,
// usage, balance_change, and none.
func NormalizeUserActivityFilter(raw string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "":
		return "", true
	case UserActivityFilterAny, "active", "has_activity":
		return UserActivityFilterAny, true
	case UserActivityFilterUsage, "used":
		return UserActivityFilterUsage, true
	case UserActivityFilterBalanceChange, "balance", "balance_changed":
		return UserActivityFilterBalanceChange, true
	case UserActivityFilterNone, "unused", "no_activity":
		return UserActivityFilterNone, true
	default:
		return "", false
	}
}

// UserActivitySearchToken returns the reserved search token consumed by the
// repository. Keeping this encoding internal lets the existing user-list
// service interface remain backward compatible while exposing a first-class
// activity query parameter at the HTTP API boundary.
func UserActivitySearchToken(activity string) string {
	normalized, ok := NormalizeUserActivityFilter(activity)
	if !ok || normalized == "" {
		return ""
	}
	return "@activity:" + normalized
}
