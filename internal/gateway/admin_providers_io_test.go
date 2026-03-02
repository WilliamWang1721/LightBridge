package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"lightbridge/internal/testutil"
	"lightbridge/internal/types"
)

func TestHandleProvidersExportAPI(t *testing.T) {
	ctx := context.Background()
	st, _ := testutil.NewStore(t)
	if err := st.UpsertProvider(ctx, types.Provider{
		ID:          "demo-provider",
		DisplayName: "Demo Provider",
		GroupName:   "test",
		Type:        types.ProviderTypeModule,
		Protocol:    types.ProtocolHTTPOpenAI,
		Endpoint:    "http://127.0.0.1:39999",
		ConfigJSON:  `{"api_key":"demo"}`,
		Enabled:     true,
		Health:      "healthy",
	}); err != nil {
		t.Fatalf("upsert provider: %v", err)
	}

	srv := &Server{store: st}
	req := httptest.NewRequest(http.MethodGet, "/admin/api/providers/export", nil)
	rr := httptest.NewRecorder()
	srv.handleProvidersExportAPI(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("content-disposition"); !strings.Contains(got, "lightbridge-providers-") {
		t.Fatalf("expected content-disposition with filename, got %q", got)
	}

	var payload struct {
		Providers []providerIOPayload `json:"providers"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode export payload: %v", err)
	}
	found := false
	for _, item := range payload.Providers {
		if strings.TrimSpace(item.ID) == "demo-provider" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("export payload missing demo-provider; got %d providers", len(payload.Providers))
	}
}

func TestHandleProvidersImportAPIReplace(t *testing.T) {
	ctx := context.Background()
	st, _ := testutil.NewStore(t)
	if err := st.UpsertProvider(ctx, types.Provider{
		ID:          "legacy-provider",
		DisplayName: "Legacy",
		Type:        types.ProviderTypeModule,
		Protocol:    types.ProtocolHTTPOpenAI,
		Endpoint:    "http://127.0.0.1:39001",
		ConfigJSON:  `{"api_key":"legacy"}`,
		Enabled:     true,
		Health:      "healthy",
	}); err != nil {
		t.Fatalf("upsert legacy provider: %v", err)
	}

	srv := &Server{store: st}
	reqBody := `{
		"replace": true,
		"providers": [
			{
				"id": "imported-provider",
				"displayName": "Imported Provider",
				"groupName": "group-a",
				"type": "module",
				"protocol": "http_openai",
				"endpoint": "http://127.0.0.1:39002",
				"configJSON": "{\"api_key\":\"imported\"}",
				"enabled": true,
				"health": "healthy"
			}
		]
	}`
	req := httptest.NewRequest(http.MethodPost, "/admin/api/providers/import", strings.NewReader(reqBody))
	req.Header.Set("content-type", "application/json")
	rr := httptest.NewRecorder()
	srv.handleProvidersImportAPI(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	imported, err := st.GetProvider(ctx, "imported-provider")
	if err != nil {
		t.Fatalf("get imported provider: %v", err)
	}
	if imported == nil {
		t.Fatalf("expected imported-provider to exist")
	}
	legacy, err := st.GetProvider(ctx, "legacy-provider")
	if err != nil {
		t.Fatalf("get legacy provider: %v", err)
	}
	if legacy != nil {
		t.Fatalf("expected legacy-provider to be removed in replace mode")
	}
}
