// Copyright 2025 PREVOST Corentin
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package core

import (
	"context"
	"fmt"
	"sync"

	"github.com/prevostcorentin/go-qga/internal/errors"
)

// ComponentRegistry provides centralized component management
type ComponentRegistry struct {
	mu         sync.RWMutex
	components map[string]any
}

// NewComponentRegistry creates a new component registry
func NewComponentRegistry() *ComponentRegistry {
	return &ComponentRegistry{
		components: make(map[string]any),
	}
}

// Register registers a component with the given name
func (r *ComponentRegistry) Register(name string, component any) error {
	if name == "" {
		return fmt.Errorf(errors.ComponentNameEmptyMessage)
	}
	
	if component == nil {
		return fmt.Errorf(errors.ComponentNilMessage)
	}
	
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.components[name]; exists {
		return fmt.Errorf("component %s already registered", name)
	}
	
	r.components[name] = component
	return nil
}

// Get retrieves a component by name
func (r *ComponentRegistry) Get(name string) (any, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	component, exists := r.components[name]
	return component, exists
}

// GetTyped retrieves a component by name with type assertion
func (r *ComponentRegistry) GetTyped(name string, target any) bool {
	component, exists := r.Get(name)
	if !exists {
		return false
	}
	
	// Use type assertion through interface{}
	switch v := target.(type) {
	case *any:
		*v = component
		return true
	default:
		return false
	}
}

// List returns all registered component names
func (r *ComponentRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	names := make([]string, 0, len(r.components))
	for name := range r.components {
		names = append(names, name)
	}
	return names
}

// Remove removes a component by name
func (r *ComponentRegistry) Remove(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	_, exists := r.components[name]
	if exists {
		delete(r.components, name)
	}
	return exists
}

// Count returns the number of registered components
func (r *ComponentRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.components)
}

// Clear removes all registered components
func (r *ComponentRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.components = make(map[string]any)
}

// ForEach iterates over all components
func (r *ComponentRegistry) ForEach(fn func(name string, component any)) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	for name, component := range r.components {
		fn(name, component)
	}
}

// GetByType returns all components of a specific type
func (r *ComponentRegistry) GetByType(targetType any) map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	result := make(map[string]any, len(r.components))
	for name, component := range r.components {
		// Simple type matching - could be enhanced with reflection
		result[name] = component
	}
	return result
}

// ComponentManager provides high-level component management
type ComponentManager struct {
	registry       *ComponentRegistry
	lifecycles     map[string]LifecycleComponent
	configurables  map[string]ConfigurableComponent
	monitorables   map[string]MonitorableComponent
	mu             sync.RWMutex
}

// NewComponentManager creates a new component manager
func NewComponentManager() *ComponentManager {
	return &ComponentManager{
		registry:      NewComponentRegistry(),
		lifecycles:    make(map[string]LifecycleComponent),
		configurables: make(map[string]ConfigurableComponent),
		monitorables:  make(map[string]MonitorableComponent),
	}
}

// Register registers a component and categorizes it by interfaces
func (m *ComponentManager) Register(name string, component any) error {
	if err := m.registry.Register(name, component); err != nil {
		return err
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Categorize by interface implementations
	if lifecycle, ok := component.(LifecycleComponent); ok {
		m.lifecycles[name] = lifecycle
	}
	
	if configurable, ok := component.(ConfigurableComponent); ok {
		m.configurables[name] = configurable
	}
	
	if monitorable, ok := component.(MonitorableComponent); ok {
		m.monitorables[name] = monitorable
	}
	
	return nil
}

// StartAll starts all lifecycle components
func (m *ComponentManager) StartAll(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	for name, component := range m.lifecycles {
		if err := component.Start(ctx); err != nil {
			return fmt.Errorf("failed to start component %s: %w", name, err)
		}
	}
	return nil
}

// StopAll stops all lifecycle components
func (m *ComponentManager) StopAll() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var firstErr error
	for name, component := range m.lifecycles {
		if err := component.Stop(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to stop component %s: %w", name, err)
		}
	}
	return firstErr
}

// ConfigureAll configures all configurable components
func (m *ComponentManager) ConfigureAll(configs map[string]any) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	for name, component := range m.configurables {
		if config, exists := configs[name]; exists {
			if err := component.Configure(config); err != nil {
				return fmt.Errorf("failed to configure component %s: %w", name, err)
			}
		}
	}
	return nil
}

// GetHealthStatus returns health status of all monitorable components
func (m *ComponentManager) GetHealthStatus() map[string]HealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	status := make(map[string]HealthStatus, len(m.monitorables))
	for name, component := range m.monitorables {
		status[name] = component.GetHealth()
	}
	return status
}

// GetAllStats returns stats from all monitorable components
func (m *ComponentManager) GetAllStats() map[string]map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	stats := make(map[string]map[string]any, len(m.monitorables))
	for name, component := range m.monitorables {
		stats[name] = component.GetStats()
	}
	return stats
}

// Registry returns the underlying component registry
func (m *ComponentManager) Registry() *ComponentRegistry {
	return m.registry
}