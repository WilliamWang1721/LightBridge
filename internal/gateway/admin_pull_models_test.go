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

func TestHandleProviderPullModelsAPIAutoRoutesUnmappedModels(t *testing.T) {
	ctx := context.Background()
	st, _ := testutil.NewStore(t)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("content-type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "new-claude-model"},
				{"id": "existing-model"},
			},
		})
	}))
	defer upstream.Close()

	cfgBytes, _ := json.Marshal(map[string]any{"base_url": upstream.URL})
	if err := st.UpsertProvider(ctx, types.Provider{
		ID:         "kiro",
		Type:       types.ProviderTypeModule,
		Protocol:   types.ProtocolHTTPOpenAI,
		Endpoint:   upstream.URL,
		ConfigJSON: string(cfgBytes),
		Enabled:    true,
		Health:     "healthy",
	}); err != nil {
		t.Fatalf("upsert kiro provider: %v", err)
	}

	if err := st.UpsertModel(ctx, types.Model{
		ID:          "existing-model",
		DisplayName: "existing-model",
		Enabled:     true,
	}); err != nil {
		t.Fatalf("upsert existing model: %v", err)
	}
	if err := st.ReplaceModelRoutes(ctx, "existing-model", []types.ModelRoute{{
		ModelID:       "existing-model",
		ProviderID:    "forward",
		UpstreamModel: "existing-upstream",
		Priority:      5,
		Weight:        1,
		Enabled:       true,
	}}); err != nil {
		t.Fatalf("replace existing model routes: %v", err)
	}

	srv := &Server{store: st}
	req := httptest.NewRequest(http.MethodPost, "/admin/api/providers/pull_models", strings.NewReader(`{"provider_id":"kiro"}`))
	req.Header.Set("content-type", "application/json")
	rr := httptest.NewRecorder()
	srv.handleProviderPullModelsAPI(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got := int(payload["auto_routed"].(float64)); got != 1 {
		t.Fatalf("expected auto_routed=1, got %d payload=%v", got, payload)
	}

	newRoutes, err := st.ListModelRoutes(ctx, "new-claude-model", true)
	if err != nil {
		t.Fatalf("list new model routes: %v", err)
	}
	if len(newRoutes) != 1 {
		t.Fatalf("expected 1 route for new model, got %d", len(newRoutes))
	}
	if newRoutes[0].ProviderID != "kiro" || newRoutes[0].UpstreamModel != "new-claude-model" {
		t.Fatalf("unexpected new model route: %+v", newRoutes[0])
	}

	existingRoutes, err := st.ListModelRoutes(ctx, "existing-model", true)
	if err != nil {
		t.Fatalf("list existing model routes: %v", err)
	}
	if len(existingRoutes) != 1 {
		t.Fatalf("expected existing routes untouched (1), got %d", len(existingRoutes))
	}
	if existingRoutes[0].ProviderID != "forward" || existingRoutes[0].UpstreamModel != "existing-upstream" || existingRoutes[0].Priority != 5 {
		t.Fatalf("existing routes changed unexpectedly: %+v", existingRoutes[0])
	}
}
