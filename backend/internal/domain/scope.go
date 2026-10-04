package domain

import (
	"fmt"
	"strings"
)

// A nil scope preserves the legacy unrestricted operator instance. An explicit
// non-nil empty scope denies every collection; brands cannot widen server scope.
type CollectionScope []string

func ParseCollectionScope(single, multiple string) (CollectionScope, error) {
	if single != "" && multiple != "" {
		return nil, fmt.Errorf("configure VHNESS_COLLECTION or VHNESS_COLLECTIONS, not both")
	}
	if single != "" {
		if !ValidID(single) {
			return nil, fmt.Errorf("invalid instance collection")
		}
		return CollectionScope{single}, nil
	}
	if multiple == "" {
		return nil, nil
	}
	var scope CollectionScope
	seen := map[string]bool{}
	for _, part := range strings.Split(multiple, ",") {
		id := strings.TrimSpace(part)
		if !ValidID(id) || seen[id] || len(scope) >= 64 {
			return nil, fmt.Errorf("invalid instance collection allowlist")
		}
		seen[id] = true
		scope = append(scope, id)
	}
	return scope, nil
}
func (s CollectionScope) Allows(id string) bool {
	if !ValidID(id) {
		return false
	}
	if s == nil {
		return true
	}
	for _, allowed := range s {
		if allowed == id {
			return true
		}
	}
	return false
}
func (s CollectionScope) Default() string {
	if s == nil {
		return DemocracyNow
	}
	if len(s) == 1 {
		return s[0]
	}
	return ""
}
