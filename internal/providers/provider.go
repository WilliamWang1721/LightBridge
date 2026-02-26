package providers

import (
	"context"
	"net/http"

	"lightbridge/internal/types"
)

type Adapter interface {
	Protocol() string
	Handle(ctx context.Context, w http.ResponseWriter, req *http.Request, provider types.Provider, route *types.ResolvedRoute) (status int, errorCode string, err error)
}

type Registry struct {
	adapters map[string]Adapter
}

func NewRegistry(adapters ...Adapter) *Registry {
	m := make(map[string]Adapter, len(adapters))
	for _, a := range adapters {
		m[a.Protocol()] = a
	}
	return &Registry{adapters: m}
}

func (r *Registry) Get(protocol string) (Adapter, bool) {
	a, ok := r.adapters[protocol]
	return a, ok
}
