package types

import "time"

const (
	ProtocolForward    = "forward"
	ProtocolAnthropic  = "anthropic"
	ProtocolGemini     = "gemini"
	ProtocolHTTPOpenAI = "http_openai"
	ProtocolHTTPRPC    = "http_rpc"
	ProtocolGRPCChat   = "grpc_chat"
	ProtocolCodex      = "codex"

	ProviderTypeBuiltin = "builtin"
	ProviderTypeModule  = "module"
)

type Provider struct {
	ID          string
	DisplayName string
	GroupName   string
	Type        string
	Protocol    string
	Endpoint    string
	ConfigJSON  string
	Enabled     bool
	Health      string
	LastCheckAt *time.Time
}

type Model struct {
	ID          string
	DisplayName string
	Enabled     bool
}

type ModelRoute struct {
	ModelID       string
	ProviderID    string
	UpstreamModel string
	Priority      int
	Weight        int
	Enabled       bool
}

type ResolvedRoute struct {
	RequestedModel string
	ProviderID     string
	UpstreamModel  string
	Variant        bool
}

type ModuleInstalled struct {
	ID          string
	Version     string
	InstallPath string
	Enabled     bool
	Protocols   string
	SHA256      string
	InstalledAt time.Time
}

type ModuleRuntime struct {
	ModuleID    string
	PID         int
	HTTPPort    int
	GRPCPort    int
	Status      string
	LastStartAt time.Time
}

type ClientAPIKey struct {
	ID         string
	Key        string
	Name       string
	Enabled    bool
	CreatedAt  time.Time
	LastUsedAt *time.Time
}

type RequestLogMeta struct {
	ID           int64
	Timestamp    time.Time
	RequestID    string
	ClientKeyID  string
	ProviderID   string
	ModelID      string
	Path         string
	Status       int
	LatencyMS    int64
	InputTokens  int
	OutputTokens int
	ErrorCode    string
}

type VirtualModelListing struct {
	ModelID      string `json:"id"`
	Object       string `json:"object"`
	Created      int64  `json:"created"`
	OwnedBy      string `json:"owned_by"`
	ProviderHint string `json:"provider_hint,omitempty"`
}

// Module index and manifest definitions.
type ModuleIndex struct {
	GeneratedAt    string        `json:"generated_at"`
	MinCoreVersion string        `json:"min_core_version"`
	Modules        []ModuleEntry `json:"modules"`
}

type ModuleEntry struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	License     string   `json:"license"`
	Tags        []string `json:"tags"`
	Protocols   []string `json:"protocols"`
	DownloadURL string   `json:"download_url"`
	SHA256      string   `json:"sha256"`
	Homepage    string   `json:"homepage"`
}

type ModuleManifest struct {
	ID             string                        `json:"id"`
	Name           string                        `json:"name"`
	Description    string                        `json:"description"`
	Tags           []string                      `json:"tags"`
	Version        string                        `json:"version"`
	License        string                        `json:"license"`
	MinCoreVersion string                        `json:"min_core_version"`
	Entrypoints    map[string]ManifestEntrypoint `json:"entrypoints"`
	Services       []ManifestService             `json:"services"`
	ConfigSchema   map[string]any                `json:"config_schema"`
	ConfigDefaults map[string]any                `json:"config_defaults"`
}

type ManifestEntrypoint struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

type ManifestService struct {
	Kind                  string         `json:"kind"`
	Protocol              string         `json:"protocol"`
	Health                ManifestHealth `json:"health"`
	ExposeProviderAliases []string       `json:"expose_provider_aliases"`
}

type ManifestHealth struct {
	Type string `json:"type"`
	Path string `json:"path"`
}
