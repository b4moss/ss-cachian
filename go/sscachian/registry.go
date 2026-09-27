package sscachian

import (
	"context"
	"fmt"
	"sync"
)

// Registry holds named KeyBuilders and Loaders for config-driven LoadTypes.
type Registry struct {
	mu          sync.RWMutex
	keyBuilders map[string]KeyBuilder
	loaders     map[string]any // Loader[T] erased
}

// NewRegistry returns a Registry with built-in KeyBuilder name "default".
func NewRegistry() *Registry {
	r := &Registry{
		keyBuilders: make(map[string]KeyBuilder),
		loaders:     make(map[string]any),
	}
	r.keyBuilders["default"] = DefaultKeyBuilder
	return r
}

// RegisterKeyBuilder binds a name. Empty name or nil fn is an error. Re-register is last-wins.
func (r *Registry) RegisterKeyBuilder(name string, kb KeyBuilder) error {
	if name == "" {
		return fmt.Errorf("%w: empty key builder name", ErrInvalidConfig)
	}
	if kb == nil {
		return fmt.Errorf("%w: nil key builder", ErrInvalidConfig)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.keyBuilders[name] = kb
	return nil
}

// RegisterLoader binds a named Loader[T]. Prefer RegisterLoaderFunc via typed helper.
// Re-register is last-wins.
func RegisterLoader[T any](r *Registry, name string, loader Loader[T]) error {
	if r == nil {
		return fmt.Errorf("%w: nil registry", ErrInvalidConfig)
	}
	if name == "" {
		return fmt.Errorf("%w: empty loader name", ErrInvalidConfig)
	}
	if loader == nil {
		return fmt.Errorf("%w: nil loader", ErrInvalidConfig)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.loaders[name] = loader
	return nil
}

// RegisterLoader is a convenience for string loaders in tests; prefer RegisterLoader[T].
func (r *Registry) RegisterLoader(name string, loader Loader[string]) error {
	return RegisterLoader(r, name, loader)
}

func (r *Registry) lookupKeyBuilder(name string) (KeyBuilder, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	kb, ok := r.keyBuilders[name]
	return kb, ok
}

func (r *Registry) lookupLoader(name string) (any, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	l, ok := r.loaders[name]
	return l, ok
}

// DriverFactory builds a Layer from config options (string values only).
type DriverFactory func(ctx context.Context, opts map[string]string) (Layer, error)

var (
	driverMu        sync.RWMutex
	driverFactories = map[string]DriverFactory{}
)

// RegisterDriverFactory registers a config driver (called from driver packages' init).
func RegisterDriverFactory(name string, f DriverFactory) {
	if name == "" || f == nil {
		return
	}
	driverMu.Lock()
	defer driverMu.Unlock()
	driverFactories[name] = f
}

func lookupDriverFactory(name string) (DriverFactory, bool) {
	driverMu.RLock()
	defer driverMu.RUnlock()
	f, ok := driverFactories[name]
	return f, ok
}
