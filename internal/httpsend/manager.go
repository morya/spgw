package httpsend

import "fmt"

type Manager struct {
	adapters map[string]Adapter
}

func NewManager() *Manager {
	return &Manager{
		adapters: make(map[string]Adapter),
	}
}

func (m *Manager) Register(adapter Adapter) {
	m.adapters[adapter.Vendor()] = adapter
}

func (m *Manager) Get(vendor string) (Adapter, error) {
	adapter, ok := m.adapters[vendor]
	if !ok {
		return nil, fmt.Errorf("adapter not found for vendor: %s", vendor)
	}
	return adapter, nil
}
