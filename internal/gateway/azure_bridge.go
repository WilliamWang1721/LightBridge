package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"lightbridge/internal/providers"
	"lightbridge/internal/types"
)

func shouldBridgeAzureLegacy(ingressProtocol, endpointKind, providerProtocol string) bool {
	return types.NormalizeProtocol(ingressProtocol) == types.ProtocolAzureOpenAI &&
		strings.HasPrefix(strings.TrimSpace(endpointKind), "azure_legacy") &&
		types.NormalizeProtocol(providerProtocol) != types.ProtocolAzureOpenAI
}

func (s *Server) handleAzureLegacyBridge(
	ctx context.Context,
	w http.ResponseWriter,
	req *http.Request,
	adapter providers.Adapter,
	provider types.Provider,
	route *types.ResolvedRoute,
	rawBody []byte,
) (int, string, error) {
	v1Path, deployment, err := azureLegacyPathToV1(req.URL.Path)
	if err != nil {
		writeOpenAIError(w, http.StatusBadRequest, err.Error(), "invalid_request_error", "invalid_legacy_path")
		return http.StatusBadRequest, "invalid_legacy_path", nil
	}

	bodyWithModel := setModelForAzureLegacyBody(rawBody, route, deployment)
	req2 := req.Clone(ctx)
	req2.URL.Path = v1Path
	req2.Body = ioNopCloser(bodyWithModel)
	req2.ContentLength = int64(len(bodyWithModel))

	status, code, callErr := adapter.Handle(ctx, w, req2, provider, route)
	if callErr != nil {
		return statusOrDefault(status, http.StatusBadGateway), code, callErr
	}
	return statusOrDefault(status, http.StatusOK), code, nil
}

func azureLegacyPathToV1(path string) (string, string, error) {
	p := strings.TrimSpace(path)
	p = strings.TrimPrefix(p, "/azure")
	const prefix = "/openai/deployments/"
	if !strings.HasPrefix(p, prefix) {
		return "", "", fmt.Errorf("invalid azure legacy path")
	}
	rest := strings.TrimPrefix(p, prefix)
	idx := strings.Index(rest, "/")
	if idx <= 0 || idx >= len(rest)-1 {
		return "", "", fmt.Errorf("invalid azure legacy path")
	}
	deployment := strings.TrimSpace(rest[:idx])
	suffix := strings.TrimPrefix(rest[idx:], "/")
	if strings.TrimSpace(suffix) == "" {
		return "", "", fmt.Errorf("invalid azure legacy path")
	}
	return "/v1/" + suffix, deployment, nil
}

func setModelForAzureLegacyBody(raw []byte, route *types.ResolvedRoute, deployment string) []byte {
	if len(bytes.TrimSpace(raw)) == 0 {
		return raw
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return raw
	}
	model := ""
	if route != nil {
		model = strings.TrimSpace(route.UpstreamModel)
	}
	if model == "" {
		model = strings.TrimSpace(deployment)
	}
	if model == "" {
		return raw
	}
	payload["model"] = model
	out, err := json.Marshal(payload)
	if err != nil {
		return raw
	}
	return out
}
