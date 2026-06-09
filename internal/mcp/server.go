// SPDX-License-Identifier: Apache-2.0

package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"cuelang.org/go/cue"
	"github.com/complytime/complypack/internal/config"
	"github.com/complytime/complypack/internal/evaluator"
	"github.com/complytime/complypack/internal/registry"
	"github.com/complytime/complypack/schemas"
	"github.com/gemaraproj/go-gemara"
	"github.com/gemaraproj/go-gemara/bundle"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/memory"
)

// Server wraps the MCP SDK server with ComplyPack-specific state.
type Server struct {
	mcp           *mcp.Server
	ResourceStore *ResourceStore
}

// ServerOptions configures ComplyPack MCP server initialization.
type ServerOptions struct {
	// ConfigPath is the path to complypack.yaml.
	ConfigPath string

	// OCIStore is the directory for OCI artifact caching.
	OCIStore string

	// CacheDir is the directory for MCP server caching.
	CacheDir string

	// EvaluatorRegistry provides available policy evaluators.
	// If nil, defaults to evaluator.DefaultRegistry().
	EvaluatorRegistry *evaluator.Registry
}

// NewServer creates a ComplyPack MCP server.
// It loads the config, reads catalogs from local paths, loads platform schemas,
// validates the platform, and creates the MCP server with resource handlers.
//
// Fails fast if:
// - Config file cannot be loaded or parsed
// - Any catalog file cannot be read
// - Platform is not supported
// - Duplicate catalog names are detected
func NewServer(ctx context.Context, opts *ServerOptions) (*Server, error) {
	if opts == nil {
		return nil, fmt.Errorf("ServerOptions cannot be nil")
	}

	// Load config
	cfg, err := config.LoadConfig(opts.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	if err := cfg.ValidateForMCP(); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Load Gemara artifacts from all configured sources
	loaded := &LoadedArtifacts{
		Catalogs: make(map[string]*gemara.ControlCatalog),
		Policies: make(map[string]*gemara.Policy),
		Guidance: make(map[string]*gemara.GuidanceCatalog),
	}
	for _, entry := range cfg.Gemara.Sources {
		src, err := loadArtifacts(ctx, entry.Source, entry.PlainHTTP)
		if err != nil {
			return nil, fmt.Errorf("failed to load artifacts from %s: %w", entry.Source, err)
		}
		if err := loaded.Merge(src); err != nil {
			return nil, fmt.Errorf("failed to merge artifacts from %s: %w", entry.Source, err)
		}
	}

	// Resolve effective policies. All loaded artifacts are intermediate
	// state consumed here — only the resolved graphs reach the runtime.
	var allCatalogs []gemara.ControlCatalog
	for _, c := range loaded.Catalogs {
		allCatalogs = append(allCatalogs, *c)
	}
	var allGuidance []gemara.GuidanceCatalog
	for _, gc := range loaded.Guidance {
		allGuidance = append(allGuidance, *gc)
	}
	effectivePolicies := make(map[string]*gemara.EffectivePolicy)
	for id, policy := range loaded.Policies {
		if len(allCatalogs) > 0 || len(allGuidance) > 0 {
			effective, err := gemara.ResolveEffectivePolicy(*policy, allCatalogs, allGuidance)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve effective policy %s: %w", id, err)
			}
			effectivePolicies[id] = effective
		}
	}

	// Build unified artifact map for MCP resource serving (marshal on demand)
	allArtifacts := make(map[string]any)
	for id, c := range loaded.Catalogs {
		allArtifacts[id] = c
	}
	for id, gc := range loaded.Guidance {
		allArtifacts[id] = gc
	}
	for id, p := range loaded.Policies {
		allArtifacts[id] = p
	}

	// Load schemas from configured sources (both bytes and compiled CUE)
	schemaMap, cueSchemaMap, err := loadSchemas(ctx, cfg.Schemas)
	if err != nil {
		return nil, fmt.Errorf("failed to load schemas: %w", err)
	}

	// Set up evaluator registry
	evalRegistry := opts.EvaluatorRegistry
	if evalRegistry == nil {
		evalRegistry = evaluator.DefaultRegistry()
	}

	store := NewResourceStore(
		allArtifacts,
		effectivePolicies,
		schemaMap,
		cueSchemaMap,
		evalRegistry,
	)

	// Create MCP server
	impl := &mcp.Implementation{
		Name:    "complypack-mcp",
		Version: "0.1.0",
	}

	mcpServer := mcp.NewServer(impl, &mcp.ServerOptions{
		Instructions: "ComplyPack MCP Server - provides Gemara catalogs and platform schemas",
	})

	// Register artifact resources
	for name := range allArtifacts {
		uri := fmt.Sprintf("%s://%s/%s", URIScheme, ResourceTypeCatalog, name)
		resource := &mcp.Resource{
			URI:      uri,
			Name:     fmt.Sprintf("Gemara Artifact: %s", name),
			MIMEType: MIMETypeYAML,
		}
		mcpServer.AddResource(resource, createResourceHandler(store, uri))
	}

	// Register schema list resource (discovery)
	schemaListURI := fmt.Sprintf("%s://%s", URIScheme, ResourceTypeSchema)
	mcpServer.AddResource(&mcp.Resource{
		URI:      schemaListURI,
		Name:     "Available Platform Schemas",
		MIMEType: MIMETypeJSON,
	}, createResourceHandler(store, schemaListURI))

	// Register per-platform schema resources
	for platform := range schemaMap {
		uri := fmt.Sprintf("%s://%s/%s", URIScheme, ResourceTypeSchema, platform)
		mime := MIMETypeCUE
		if isJSONSchema(schemaMap[platform]) {
			mime = MIMETypeJSONSchema
		}
		resource := &mcp.Resource{
			URI:      uri,
			Name:     fmt.Sprintf("Platform Schema: %s", platform),
			MIMEType: mime,
		}
		mcpServer.AddResource(resource, createResourceHandler(store, uri))
	}

	// Register evaluator resource
	evalURI := fmt.Sprintf("%s://%s", URIScheme, ResourceTypeEvaluator)
	evalResource := &mcp.Resource{
		URI:      evalURI,
		Name:     "Available Policy Evaluators",
		MIMEType: MIMETypeJSON,
	}
	mcpServer.AddResource(evalResource, createResourceHandler(store, evalURI))

	// Register tools
	validateTool := createValidatePolicyTool()
	mcpServer.AddTool(validateTool, handleValidatePolicy(store))

	testTool := createTestPolicyTool()
	mcpServer.AddTool(testTool, handleTestPolicy(store))

	assessmentTool := createGetAssessmentRequirementsTool()
	mcpServer.AddTool(assessmentTool, handleGetAssessmentRequirements(store))

	return &Server{
		mcp:           mcpServer,
		ResourceStore: store,
	}, nil
}

// createResourceHandler creates a ResourceHandler that reads from the ResourceStore.
func createResourceHandler(store *ResourceStore, uri string) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		contents, err := store.ReadResource(ctx, req.Params.URI)
		if err != nil {
			return nil, err
		}
		return &mcp.ReadResourceResult{Contents: contents}, nil
	}
}



// Run starts the MCP server on the given transport.
// It delegates to the underlying MCP SDK server's Run method.
func (s *Server) Run(ctx context.Context, transport mcp.Transport) error {
	return s.mcp.Run(ctx, transport)
}

// loadSchemas loads platform schemas from configured sources, returning both
// the raw bytes (for MCP resource serving) and compiled CUE values (for
// contract validation). Fails fast if a user-configured source cannot be
// loaded (per ADR 004). Embedded schemas are used only when no source is
// configured.
func loadSchemas(ctx context.Context, schemaRefs []config.SchemaRef) (map[string][]byte, map[string]cue.Value, error) {
	schemaMap := make(map[string][]byte)
	cueSchemaMap := make(map[string]cue.Value)

	for _, ref := range schemaRefs {
		platform := ref.Platform

		// Determine source (new field takes precedence over legacy path)
		source := ref.Source
		if source == "" && ref.Path != "" {
			source = "file://" + ref.Path
		}

		var data []byte
		var cueVal cue.Value
		var err error

		if source != "" {
			parsed, parseErr := ParseSchemaSource(source)
			if parseErr != nil {
				return nil, nil, fmt.Errorf("failed to parse schema source for %s: %w", platform, parseErr)
			}

			data, err = loadSchemaBytes(ctx, parsed, platform)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to load schema for platform %s from %s: %w", platform, source, err)
			}

			cueVal, err = LoadCUEFromSource(ctx, parsed, platform)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to load CUE schema for platform %s from %s: %w", platform, source, err)
			}

			slog.Info("loaded schema from source", "platform", platform, "source", source)
		} else {
			data, err = schemas.GetBuiltInSchema(platform)
			if err != nil {
				slog.Warn("no embedded schema available for platform, skipping",
					"platform", platform, "error", err)
				continue
			}

			cueVal, err = loadEmbeddedCUESchema(platform)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to compile embedded CUE schema for %s: %w", platform, err)
			}

			slog.Info("loaded embedded schema", "platform", platform)
		}

		schemaMap[platform] = data
		cueSchemaMap[platform] = cueVal
	}

	return schemaMap, cueSchemaMap, nil
}

// loadSchemaBytes loads schema content as bytes from a parsed source.
// For CUE modules, serializes definitions as CUE syntax (LLMs can interpret
// CUE definitions to understand field structure, types, and constraints).
// For other sources, returns the raw file content.
func loadSchemaBytes(ctx context.Context, source SchemaSource, platform string) ([]byte, error) {
	switch source.Type {
	case SourceTypeCUEModule:
		cueVal, err := LoadCUEFromSource(ctx, source, platform)
		if err != nil {
			return nil, fmt.Errorf("failed to load CUE schema: %w", err)
		}
		cueSyntax := formatCUEDefinitions(cueVal)
		if len(cueSyntax) == 0 {
			return nil, fmt.Errorf("no definitions found in CUE module for %s", platform)
		}
		return cueSyntax, nil

	case SourceTypeHTTPS, SourceTypeHTTP:
		data, format, err := fetchSchemaFromURL(ctx, source.Path)
		if err != nil {
			return nil, err
		}
		if format != FormatJSON {
			return nil, fmt.Errorf("expected JSON format, got %v", format)
		}
		return data, nil

	case SourceTypeFile, SourceTypeLegacyPath:
		data, format, err := loadSchemaFromFile(source.Path)
		if err != nil {
			return nil, err
		}
		if format != FormatJSON {
			return nil, fmt.Errorf("expected JSON format, got %v", format)
		}
		return data, nil

	case SourceTypeUnknown:
		return nil, fmt.Errorf("no source specified")

	default:
		return nil, fmt.Errorf("unsupported source type: %v", source.Type)
	}
}

// formatCUEDefinitions extracts top-level definitions from a CUE value and
// formats them as readable CUE syntax for LLM consumption.
func formatCUEDefinitions(val cue.Value) []byte {
	var sb strings.Builder

	iter, _ := val.Fields(cue.Definitions(true))
	for iter.Next() {
		label := iter.Selector().String()
		defVal := iter.Value()
		fmt.Fprintf(&sb, "%s: %v\n\n", label, defVal)
	}

	return []byte(sb.String())
}

// LoadedArtifacts holds parsed artifacts from bundle/file loading.
// All fields are intermediate — consumed during effective policy
// resolution in NewServer and not passed to ResourceStore.
type LoadedArtifacts struct {
	Catalogs map[string]*gemara.ControlCatalog
	Policies map[string]*gemara.Policy
	Guidance map[string]*gemara.GuidanceCatalog
}

// Merge combines another LoadedArtifacts into this one.
// Returns an error if any artifact ID appears in both.
func (la *LoadedArtifacts) Merge(other *LoadedArtifacts) error {
	for id, cat := range other.Catalogs {
		if _, exists := la.Catalogs[id]; exists {
			return fmt.Errorf("duplicate artifact id %q across sources", id)
		}
		la.Catalogs[id] = cat
	}
	for id, pol := range other.Policies {
		if _, exists := la.Policies[id]; exists {
			return fmt.Errorf("duplicate artifact id %q across sources", id)
		}
		la.Policies[id] = pol
	}
	for id, gc := range other.Guidance {
		la.Guidance[id] = gc
	}
	return nil
}

// loadArtifacts loads and classifies Gemara artifacts from either a file path or OCI reference.
// For OCI references, it returns both the primary artifact and any imports (bundle).
// For file paths, it returns the single artifact.
func loadArtifacts(ctx context.Context, source string, plainHTTP bool) (*LoadedArtifacts, error) {
	// Parse URI scheme
	if strings.HasPrefix(source, "file://") {
		// file:// URI - strip scheme and load local file
		path := strings.TrimPrefix(source, "file://")
		return loadFileArtifacts(ctx, path)
	}

	if strings.HasPrefix(source, "oci://") {
		// oci:// URI - strip scheme and pull from OCI registry
		ref := strings.TrimPrefix(source, "oci://")
		return loadBundleArtifacts(ctx, ref, plainHTTP)
	}

	// Legacy: No scheme - detect OCI vs file path
	if isOCIReference(source) {
		// Pull from OCI registry - returns primary + imports
		return loadBundleArtifacts(ctx, source, plainHTTP)
	}

	// Load from local file path - single artifact
	return loadFileArtifacts(ctx, source)
}

// loadFileArtifacts loads and classifies a single artifact from a file.
func loadFileArtifacts(ctx context.Context, path string) (*LoadedArtifacts, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Classify the artifact using gemara.Classify
	artifactSet, err := gemara.Classify(data)
	if err != nil {
		return nil, fmt.Errorf("failed to classify artifact: %w", err)
	}

	result := &LoadedArtifacts{
		Catalogs: make(map[string]*gemara.ControlCatalog),
		Policies: make(map[string]*gemara.Policy),
		Guidance: make(map[string]*gemara.GuidanceCatalog),
	}

	for _, catalog := range artifactSet.ControlCatalogs {
		result.Catalogs[catalog.Metadata.Id] = &catalog
	}
	for _, gc := range artifactSet.GuidanceCatalogs {
		gc := gc
		result.Guidance[gc.Metadata.Id] = &gc
	}
	for _, policy := range artifactSet.Policies {
		result.Policies[policy.Metadata.Id] = &policy
	}

	return result, nil
}

// loadBundleArtifacts loads and classifies artifacts from an OCI bundle.
func loadBundleArtifacts(ctx context.Context, ref string, plainHTTP bool) (*LoadedArtifacts, error) {
	// Get Docker credentials
	credFunc, err := registry.NewCredentialFunc()
	if err != nil {
		return nil, fmt.Errorf("failed to load Docker credentials: %w", err)
	}

	// Create remote repository
	repo, err := registry.NewRepository(ref, credFunc, plainHTTP)
	if err != nil {
		return nil, err
	}

	// Extract tag from reference
	tag := registry.ParseTag(ref)

	// Resolve and pull manifest
	store := memory.New()
	_, err = oras.Copy(ctx, repo, tag, store, tag, oras.DefaultCopyOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to pull from registry: %w", err)
	}

	// Unpack the Gemara bundle
	b, err := bundle.Unpack(ctx, store, tag)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack bundle: %w", err)
	}

	// Classify the bundle
	classified, err := b.Classify()
	if err != nil {
		return nil, fmt.Errorf("failed to classify bundle: %w", err)
	}

	result := &LoadedArtifacts{
		Catalogs: make(map[string]*gemara.ControlCatalog),
		Policies: make(map[string]*gemara.Policy),
		Guidance: make(map[string]*gemara.GuidanceCatalog),
	}

	if classified.Policy != nil {
		result.Policies[classified.Policy.Metadata.Id] = classified.Policy
	}
	if classified.ControlCatalog != nil {
		result.Catalogs[classified.ControlCatalog.Metadata.Id] = classified.ControlCatalog
	}
	if classified.GuidanceCatalog != nil {
		result.Guidance[classified.GuidanceCatalog.Metadata.Id] = classified.GuidanceCatalog
	}

	if classified.Imports != nil {
		for _, catalog := range classified.Imports.ControlCatalogs {
			result.Catalogs[catalog.Metadata.Id] = &catalog
		}
		for _, gc := range classified.Imports.GuidanceCatalogs {
			gc := gc
			result.Guidance[gc.Metadata.Id] = &gc
		}
	}

	return result, nil
}

// isOCIReference returns true if the source looks like an OCI reference.
func isOCIReference(source string) bool {
	// OCI references contain a registry host (domain with optional port)
	// Examples: ghcr.io/org/repo:tag, localhost:5000/repo:tag, http://registry/repo
	return strings.Contains(source, "/") && (strings.Contains(source, ":") || strings.Contains(source, "//"))
}

// pullCatalogsFromOCI pulls a Gemara catalog and its imports from an OCI registry.
// Returns a map of catalog name -> catalog data (YAML bytes).
