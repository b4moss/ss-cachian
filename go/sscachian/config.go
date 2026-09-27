package sscachian

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const configSchemaVersion = "0.9"

// file-level config DTOs (YAML/JSON).
type rawFile struct {
	SchemaVersion any       `json:"schema_version" yaml:"schema_version"`
	Types         []rawType `json:"types" yaml:"types"`
}

type rawType struct {
	Name       string     `json:"name" yaml:"name"`
	KeyBuilder string     `json:"key_builder" yaml:"key_builder"`
	Loader     string     `json:"loader" yaml:"loader"`
	TTL        any        `json:"ttl" yaml:"ttl"`
	Layers     []rawLayer `json:"layers" yaml:"layers"`
}

type rawLayer struct {
	Driver  string         `json:"driver" yaml:"driver"`
	TTL     any            `json:"ttl" yaml:"ttl"`
	Options map[string]any `json:"options" yaml:"options"`
}

// LoadTypes loads Cache Types from a YAML or JSON file path (by extension).
// Driver factories are registered by importing driver packages
// (e.g. blank-import github.com/b4moss/ss-cachian/driver/memory and .../firestore).
func LoadTypes[T any](ctx context.Context, reg *Registry, path string) (map[string]*CacheType[T], error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		return LoadTypesJSON[T](ctx, reg, data)
	case ".yaml", ".yml":
		return LoadTypesYAML[T](ctx, reg, data)
	default:
		return nil, fmt.Errorf("%w: unsupported config extension %q", ErrInvalidConfig, ext)
	}
}

// LoadTypesYAML parses YAML bytes and builds Cache Types.
func LoadTypesYAML[T any](ctx context.Context, reg *Registry, data []byte) (map[string]*CacheType[T], error) {
	var raw rawFile
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%w: yaml: %v", ErrInvalidConfig, err)
	}
	return buildTypes[T](ctx, reg, raw)
}

// LoadTypesJSON parses JSON bytes and builds Cache Types.
func LoadTypesJSON[T any](ctx context.Context, reg *Registry, data []byte) (map[string]*CacheType[T], error) {
	var raw rawFile
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%w: json: %v", ErrInvalidConfig, err)
	}
	return buildTypes[T](ctx, reg, raw)
}

// LoadTypesReader reads all of r and parses as YAML (format=yaml) or JSON (format=json).
func LoadTypesReader[T any](ctx context.Context, reg *Registry, r io.Reader, format string) (map[string]*CacheType[T], error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	switch strings.ToLower(format) {
	case "json":
		return LoadTypesJSON[T](ctx, reg, data)
	case "yaml", "yml":
		return LoadTypesYAML[T](ctx, reg, data)
	default:
		return nil, fmt.Errorf("%w: unsupported format %q", ErrInvalidConfig, format)
	}
}

func buildTypes[T any](ctx context.Context, reg *Registry, raw rawFile) (map[string]*CacheType[T], error) {
	if reg == nil {
		return nil, fmt.Errorf("%w: nil registry", ErrInvalidConfig)
	}
	ver, err := schemaVersionString(raw.SchemaVersion)
	if err != nil {
		return nil, err
	}
	if ver != configSchemaVersion {
		return nil, fmt.Errorf("%w: unsupported schema_version %q", ErrInvalidConfig, ver)
	}
	if len(raw.Types) == 0 {
		return nil, fmt.Errorf("%w: types must not be empty", ErrInvalidConfig)
	}

	out := make(map[string]*CacheType[T], len(raw.Types))
	for _, rt := range raw.Types {
		if rt.Name == "" {
			return nil, fmt.Errorf("%w: type name must not be empty", ErrInvalidConfig)
		}
		if _, exists := out[rt.Name]; exists {
			return nil, fmt.Errorf("%w: duplicate type name %q", ErrInvalidConfig, rt.Name)
		}
		ct, err := buildOneType[T](ctx, reg, rt)
		if err != nil {
			return nil, err
		}
		out[rt.Name] = ct
	}
	return out, nil
}

func buildOneType[T any](ctx context.Context, reg *Registry, rt rawType) (*CacheType[T], error) {
	if len(rt.Layers) == 0 {
		return nil, fmt.Errorf("%w: type %q: layers must not be empty", ErrInvalidConfig, rt.Name)
	}

	kbName := rt.KeyBuilder
	if kbName == "" {
		kbName = "default"
	}
	kb, ok := reg.lookupKeyBuilder(kbName)
	if !ok {
		return nil, fmt.Errorf("%w: unknown key_builder %q", ErrInvalidConfig, kbName)
	}

	var loader Loader[T]
	if rt.Loader != "" {
		rawLoader, ok := reg.lookupLoader(rt.Loader)
		if !ok {
			return nil, fmt.Errorf("%w: unknown loader %q", ErrInvalidConfig, rt.Loader)
		}
		typed, ok := rawLoader.(Loader[T])
		if !ok {
			return nil, fmt.Errorf("%w: loader %q type mismatch", ErrInvalidConfig, rt.Loader)
		}
		loader = typed
	}

	typeTTL, typeTTLSet, err := parseOptionalTTL(rt.TTL)
	if err != nil {
		return nil, fmt.Errorf("%w: type %q: %v", ErrInvalidConfig, rt.Name, err)
	}

	layers := make([]Layer, 0, len(rt.Layers))
	ttls := make([]time.Duration, 0, len(rt.Layers))
	for i, rl := range rt.Layers {
		layer, err := buildLayer(ctx, rl)
		if err != nil {
			return nil, fmt.Errorf("%w: type %q layer[%d]: %v", ErrInvalidConfig, rt.Name, i, err)
		}
		layers = append(layers, layer)

		layerTTL, layerTTLSet, err := parseOptionalTTL(rl.TTL)
		if err != nil {
			return nil, fmt.Errorf("%w: type %q layer[%d]: %v", ErrInvalidConfig, rt.Name, i, err)
		}
		switch {
		case layerTTLSet:
			ttls = append(ttls, layerTTL)
		case typeTTLSet:
			ttls = append(ttls, typeTTL)
		default:
			ttls = append(ttls, 0)
		}
	}

	b := Define[T](rt.Name).WithLayers(layers...).WithKeyBuilder(kb).WithLayerTTLs(ttls...)
	if loader != nil {
		b = b.WithLoader(loader)
	}
	return b.Build()
}

func buildLayer(ctx context.Context, rl rawLayer) (Layer, error) {
	if rl.Driver == "" {
		return nil, fmt.Errorf("driver must not be empty")
	}
	f, ok := lookupDriverFactory(rl.Driver)
	if !ok {
		return nil, fmt.Errorf("unknown driver %q", rl.Driver)
	}
	opts, err := stringifyOptions(rl.Options)
	if err != nil {
		return nil, err
	}
	return f(ctx, opts)
}

func stringifyOptions(opts map[string]any) (map[string]string, error) {
	if opts == nil {
		return map[string]string{}, nil
	}
	out := make(map[string]string, len(opts))
	for k, v := range opts {
		switch x := v.(type) {
		case string:
			out[k] = x
		case nil:
			return nil, fmt.Errorf("option %q must be a string", k)
		default:
			return nil, fmt.Errorf("option %q must be a string", k)
		}
	}
	return out, nil
}

func schemaVersionString(v any) (string, error) {
	if v == nil {
		return "", fmt.Errorf("%w: schema_version is required", ErrInvalidConfig)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("%w: schema_version must be a string", ErrInvalidConfig)
	}
	if s == "" {
		return "", fmt.Errorf("%w: schema_version is required", ErrInvalidConfig)
	}
	return s, nil
}

func parseOptionalTTL(v any) (time.Duration, bool, error) {
	if v == nil {
		return 0, false, nil
	}
	s, ok := v.(string)
	if !ok {
		return 0, false, fmt.Errorf("ttl must be a string duration")
	}
	if s == "" {
		return 0, false, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, false, fmt.Errorf("invalid ttl %q", s)
	}
	if d < 0 {
		return 0, false, ErrNegativeTTL
	}
	return d, true, nil
}
