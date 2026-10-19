package router

import (
	"odition/config"
	"strings"
	"sync"
)

type Router struct {
	mu       sync.RWMutex
	topology *config.Topology
}

func NewRouter(top *config.Topology) *Router {
	return &Router{
		topology: top,
	}
}

func (r *Router) Route(rawSMS string) (familyID string, code uint8) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.topology == nil {
		return "", 0
	}

	msg := strings.ToLower(strings.TrimSpace(rawSMS))
	for _, family := range r.topology.Families {
		for _, anchor := range family.Anchors {
			if strings.Contains(msg, strings.ToLower(anchor)) {
				return family.ID, family.Code
			}
		}
	}
	return "", 0
}
