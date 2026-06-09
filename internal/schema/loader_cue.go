// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/mod/modconfig"
	"cuelang.org/go/mod/modregistry"
)

// CUELoader loads schemas from the CUE Central Registry.
type CUELoader struct{}

func (l *CUELoader) Match(source string) bool {
	return strings.HasPrefix(source, "cue://")
}

func (l *CUELoader) Load(ctx context.Context, source string, platform string) (*Schema, error) {
	modulePath := strings.TrimPrefix(source, "cue://")
	if modulePath == "" {
		return nil, fmt.Errorf("cue:// scheme requires module path")
	}

	var fragment string
	if idx := strings.LastIndex(modulePath, "#"); idx >= 0 {
		fragment = modulePath[idx+1:]
		modulePath = modulePath[:idx]
	}

	val, err := loadFromCUERegistry(ctx, modulePath)
	if err != nil {
		return nil, err
	}

	cueVal, err := resolveCUEDefinition(val, fragment)
	if err != nil {
		return nil, err
	}

	cueSyntax := FormatCUEDefinitions(val)
	if len(cueSyntax) == 0 {
		return nil, fmt.Errorf("no definitions found in CUE module for %s", platform)
	}

	return &Schema{
		Platform: platform,
		Bytes:    cueSyntax,
		CUE:      cueVal,
	}, nil
}

func resolveCUEDefinition(val cue.Value, fragment string) (cue.Value, error) {
	if fragment != "" {
		resolved := val.LookupPath(cue.MakePath(cue.Def(fragment)))
		if !resolved.Exists() {
			return cue.Value{}, fmt.Errorf("definition #%s not found in schema", fragment)
		}
		return resolved, nil
	}

	hasRegularFields := false
	iter, _ := val.Fields(cue.Optional(true))
	for iter.Next() {
		hasRegularFields = true
		break
	}

	if hasRegularFields {
		return val, nil
	}

	var defs []string
	defIter, _ := val.Fields(cue.Definitions(true))
	for defIter.Next() {
		defs = append(defs, defIter.Selector().String())
	}

	if len(defs) == 0 {
		return cue.Value{}, fmt.Errorf("schema has no fields or definitions")
	}

	return cue.Value{}, fmt.Errorf(
		"schema has only definitions, specify one with #Fragment in the source URL (available: %s)",
		strings.Join(defs, ", "),
	)
}

func loadFromCUERegistry(ctx context.Context, modulePath string) (cue.Value, error) {
	modPath, version := SplitModuleVersion(modulePath)

	slog.Info("loading schema from CUE registry", "module", modPath, "requestedVersion", version)

	resolver, err := modconfig.NewResolver(nil)
	if err != nil {
		return cue.Value{}, fmt.Errorf("creating CUE resolver: %w", err)
	}

	if version == "" || version == "latest" {
		resolved, resolveErr := resolveLatestVersion(ctx, modPath, resolver)
		if resolveErr != nil {
			return cue.Value{}, fmt.Errorf("resolving latest version for %s: %w", modPath, resolveErr)
		}
		version = resolved
		slog.Info("resolved latest version", "module", modPath, "version", version)
	}

	tmpDir, err := os.MkdirTemp("", "complypack-cue-*")
	if err != nil {
		return cue.Value{}, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := writeCUEWorkspace(tmpDir, modPath, version); err != nil {
		return cue.Value{}, fmt.Errorf("writing temp CUE workspace: %w", err)
	}

	reg, err := modconfig.NewRegistry(nil)
	if err != nil {
		return cue.Value{}, fmt.Errorf("creating CUE registry: %w", err)
	}

	importPath := ImportPathForModule(modPath)

	instances := load.Instances([]string{importPath}, &load.Config{
		Dir:      tmpDir,
		Registry: reg,
	})
	if len(instances) == 0 {
		return cue.Value{}, fmt.Errorf("loading module %s: no instances returned", modPath)
	}
	if err := instances[0].Err; err != nil {
		return cue.Value{}, fmt.Errorf("loading module %s@%s: %w", modPath, version, err)
	}

	cueCtx := cuecontext.New()
	val := cueCtx.BuildInstance(instances[0])
	if err := val.Err(); err != nil {
		return cue.Value{}, fmt.Errorf("building schema from %s@%s: %w", modPath, version, err)
	}

	return val, nil
}

// SplitModuleVersion separates a module path from its version.
func SplitModuleVersion(input string) (string, string) {
	idx := strings.LastIndex(input, "@")
	if idx < 0 {
		return input, ""
	}

	path := input[:idx]
	version := input[idx+1:]

	if IsMajorOnly(version) {
		return input, ""
	}

	return path, version
}

// IsMajorOnly returns true if v matches "v0", "v1", "v2", etc.
func IsMajorOnly(v string) bool {
	if !strings.HasPrefix(v, "v") {
		return false
	}
	rest := v[1:]
	if rest == "" {
		return false
	}
	for _, c := range rest {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func resolveLatestVersion(ctx context.Context, modPath string, resolver *modconfig.Resolver) (string, error) {
	client := modregistry.NewClientWithResolver(resolver)

	versions, err := client.ModuleVersions(ctx, modPath)
	if err != nil {
		return "", fmt.Errorf("listing versions: %w", err)
	}
	if len(versions) == 0 {
		return "", fmt.Errorf("no versions found for %s", modPath)
	}

	return versions[len(versions)-1], nil
}

func writeCUEWorkspace(dir, modPath, version string) error {
	modDir := filepath.Join(dir, "cue.mod")
	if err := os.MkdirAll(modDir, 0755); err != nil {
		return err
	}

	depKey := modPath
	if !strings.Contains(modPath, "@") {
		depKey = modPath + "@v0"
	}

	moduleCUE := fmt.Sprintf(`module: "complypack.local/schema@v0"
language: version: "v0.16.1"
deps: "%s": v: "%s"
`, depKey, version)

	return os.WriteFile(filepath.Join(modDir, "module.cue"), []byte(moduleCUE), 0600)
}

// ImportPathForModule returns the CUE import path for a module path.
func ImportPathForModule(modPath string) string {
	if idx := strings.LastIndex(modPath, "@"); idx > 0 {
		suffix := modPath[idx+1:]
		if IsMajorOnly(suffix) {
			return modPath[:idx]
		}
	}
	return modPath
}
