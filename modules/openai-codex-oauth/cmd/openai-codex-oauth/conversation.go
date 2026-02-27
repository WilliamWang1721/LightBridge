package main

import (
	"strings"
	"time"
)

type conversationEntry struct {
	ID       string
	ExpireAt time.Time
}

func (s *server) promptCacheKeyFromSession(model, sessionKey string) string {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return ""
	}
	model = strings.TrimSpace(model)
	cacheKey := sessionKey
	if model != "" {
		cacheKey = model + ":" + sessionKey
	}

	now := time.Now()
	s.convMu.Lock()
	defer s.convMu.Unlock()

	if ent, ok := s.conversations[cacheKey]; ok {
		if ent.ID != "" && ent.ExpireAt.After(now) {
			return ent.ID
		}
	}
	ent := conversationEntry{
		ID:       newUUID(),
		ExpireAt: now.Add(1 * time.Hour),
	}
	s.conversations[cacheKey] = ent
	return ent.ID
}
