package repository

import (
	"strings"

	"github.com/WilliamWang1721/LightBridge/ent/predicate"
	entredeemcode "github.com/WilliamWang1721/LightBridge/ent/redeemcode"
	dbuser "github.com/WilliamWang1721/LightBridge/ent/user"
	"github.com/WilliamWang1721/LightBridge/internal/service"

	entsql "entgo.io/ent/dialect/sql"
)

const userActivitySearchPrefix = "@activity:"

// splitUserAdvancedSearch removes recognized reserved activity expressions from
// the ordinary fuzzy-search text. Multiple expressions are allowed; the last
// valid expression wins so clients can safely override a previously assembled
// filter.
func splitUserAdvancedSearch(raw string) (searchText, activity string) {
	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return "", ""
	}

	searchParts := make([]string, 0, len(parts))
	for _, part := range parts {
		lower := strings.ToLower(part)
		if strings.HasPrefix(lower, userActivitySearchPrefix) {
			candidate := part[len(userActivitySearchPrefix):]
			if normalized, ok := service.NormalizeUserActivityFilter(candidate); ok && normalized != "" {
				activity = normalized
				continue
			}
		}
		searchParts = append(searchParts, part)
	}
	return strings.Join(searchParts, " "), activity
}

func userActivityPredicate(activity string) predicate.User {
	usage := dbuser.HasUsageLogs()
	balanceChange := dbuser.Or(
		dbuser.HasRedeemCodesWith(entredeemcode.TypeIn(
			service.RedeemTypeBalance,
			service.AdjustmentTypeAdminBalance,
		)),
		userHasAffiliateBalanceTransfer(),
	)

	switch activity {
	case service.UserActivityFilterUsage:
		return usage
	case service.UserActivityFilterBalanceChange:
		return balanceChange
	case service.UserActivityFilterNone:
		return dbuser.Not(dbuser.Or(usage, balanceChange))
	case service.UserActivityFilterAny:
		fallthrough
	default:
		return dbuser.Or(usage, balanceChange)
	}
}

// Affiliate transfers are stored outside Ent's user edges, so use a correlated
// EXISTS predicate. This keeps filtering in SQL, preserving correct totals and
// pagination for large user sets.
func userHasAffiliateBalanceTransfer() predicate.User {
	return predicate.User(func(s *entsql.Selector) {
		s.Where(entsql.P(func(b *entsql.Builder) {
			b.WriteString("EXISTS (SELECT 1 FROM user_affiliate_ledger AS ual WHERE ual.user_id = ").
				Ident(s.C(dbuser.FieldID)).
				WriteString(" AND ual.action = ").
				Arg("transfer").
				WriteString(")")
		}))
	})
}
