//go:build unit

package service

func billingModelForRestriction(source, requestedModel, channelMappedModel string) string {
	switch source {
	case BillingModelSourceRequested:
		return requestedModel
	case BillingModelSourceUpstream:
		return ""
	case BillingModelSourceChannelMapped:
		return channelMappedModel
	default:
		return channelMappedModel
	}
}

func resolveAccountUpstreamModel(account *Account, requestedModel string) string {
	if account == nil {
		return ""
	}
	if account.IsAntigravity() {
		return mapAntigravityModel(account, requestedModel)
	}
	return account.GetMappedModel(requestedModel)
}
