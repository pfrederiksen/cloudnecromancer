package parser

import (
	"fmt"
	"sync"
)

var (
	registryMu sync.RWMutex
	parsers    = make(map[string]Parser)
)

// Register adds a parser to the global registry, mapping each of its supported events.
func Register(p Parser) {
	registryMu.Lock()
	defer registryMu.Unlock()
	for _, eventName := range p.SupportedEvents() {
		parsers[eventName] = p
	}
}

// Lookup returns the parser registered for the given CloudTrail event name.
func Lookup(eventName string) (Parser, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	p, ok := parsers[eventName]
	if !ok {
		return nil, fmt.Errorf("no parser registered for event: %s", eventName)
	}
	return p, nil
}

// RegisteredEvents returns all event names that have a registered parser.
func RegisteredEvents() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	events := make([]string, 0, len(parsers))
	for e := range parsers {
		events = append(events, e)
	}
	return events
}
