package providers

import (
	"context"
	"net/http"

	"lightbridge/internal/types"
)

type GRPCChatAdapter struct{}

func NewGRPCChatAdapter() *GRPCChatAdapter {
	return &GRPCChatAdapter{}
}

func (a *GRPCChatAdapter) Protocol() string {
	return types.ProtocolGRPCChat
}

func (a *GRPCChatAdapter) Handle(_ context.Context, w http.ResponseWriter, _ *http.Request, _ types.Provider, _ *types.ResolvedRoute) (int, string, error) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	_, _ = w.Write([]byte(`{"error":{"message":"grpc_chat adapter is reserved for module-specific integration in this MVP","type":"not_supported","code":"501_not_supported"}}`))
	return http.StatusNotImplemented, "501_not_supported", nil
}
