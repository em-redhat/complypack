// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"oras.land/oras-go/v2/registry/remote/auth"
)

func TestCredentialPaths(t *testing.T) {
	// Save and unset env vars that affect path resolution.
	// t.Setenv restores them automatically after the test.

	t.Run("default Docker path from HOME", func(t *testing.T) {
		t.Setenv(dockerConfigDirEnv, "")
		t.Setenv(xdgRuntimeDirEnv, "")

		paths := credentialPaths()
		if len(paths) < 1 {
			t.Fatal("expected at least one path (Docker default)")
		}

		homeDir, _ := os.UserHomeDir()
		wantDocker := filepath.Join(homeDir, ".docker", "config.json")
		if paths[0] != wantDocker {
			t.Errorf("paths[0] = %q, want %q", paths[0], wantDocker)
		}

		// Podman config path should be present (from HOME)
		wantPodman := filepath.Join(homeDir, ".config", "containers", "auth.json")
		found := false
		for _, p := range paths {
			if p == wantPodman {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected Podman config path %q in paths %v", wantPodman, paths)
		}
	})

	t.Run("DOCKER_CONFIG overrides default Docker path", func(t *testing.T) {
		customDir := t.TempDir()
		t.Setenv(dockerConfigDirEnv, customDir)
		t.Setenv(xdgRuntimeDirEnv, "")

		paths := credentialPaths()
		if len(paths) < 1 {
			t.Fatal("expected at least one path")
		}

		wantDocker := filepath.Join(customDir, "config.json")
		if paths[0] != wantDocker {
			t.Errorf("paths[0] = %q, want %q", paths[0], wantDocker)
		}
	})

	t.Run("XDG_RUNTIME_DIR adds Podman runtime path", func(t *testing.T) {
		runtimeDir := t.TempDir()
		t.Setenv(dockerConfigDirEnv, "")
		t.Setenv(xdgRuntimeDirEnv, runtimeDir)

		paths := credentialPaths()
		wantPodmanRuntime := filepath.Join(runtimeDir, "containers", "auth.json")
		found := false
		for _, p := range paths {
			if p == wantPodmanRuntime {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected Podman runtime path %q in paths %v", wantPodmanRuntime, paths)
		}
	})

	t.Run("XDG_RUNTIME_DIR unset omits Podman runtime path", func(t *testing.T) {
		t.Setenv(dockerConfigDirEnv, "")
		t.Setenv(xdgRuntimeDirEnv, "")

		paths := credentialPaths()
		for _, p := range paths {
			if filepath.Base(filepath.Dir(p)) == "containers" &&
				filepath.Base(p) == "auth.json" {
				// This should only be the ~/.config/containers path, not a runtime path
				if filepath.Dir(filepath.Dir(p)) != filepath.Join(mustHomeDir(t), ".config") {
					t.Errorf("unexpected Podman runtime path %q when XDG_RUNTIME_DIR is unset", p)
				}
			}
		}
	})

	t.Run("all env vars set produces three paths", func(t *testing.T) {
		dockerDir := t.TempDir()
		runtimeDir := t.TempDir()
		t.Setenv(dockerConfigDirEnv, dockerDir)
		t.Setenv(xdgRuntimeDirEnv, runtimeDir)

		paths := credentialPaths()
		if len(paths) != 3 {
			t.Fatalf("expected 3 paths, got %d: %v", len(paths), paths)
		}

		wantDocker := filepath.Join(dockerDir, "config.json")
		wantRuntime := filepath.Join(runtimeDir, "containers", "auth.json")
		homeDir := mustHomeDir(t)
		wantConfig := filepath.Join(homeDir, ".config", "containers", "auth.json")

		if paths[0] != wantDocker {
			t.Errorf("paths[0] = %q, want %q", paths[0], wantDocker)
		}
		if paths[1] != wantRuntime {
			t.Errorf("paths[1] = %q, want %q", paths[1], wantRuntime)
		}
		if paths[2] != wantConfig {
			t.Errorf("paths[2] = %q, want %q", paths[2], wantConfig)
		}
	})

	t.Run("path order is Docker then Podman runtime then Podman config", func(t *testing.T) {
		dockerDir := t.TempDir()
		runtimeDir := t.TempDir()
		t.Setenv(dockerConfigDirEnv, dockerDir)
		t.Setenv(xdgRuntimeDirEnv, runtimeDir)

		paths := credentialPaths()
		if len(paths) != 3 {
			t.Fatalf("expected 3 paths, got %d: %v", len(paths), paths)
		}

		// Docker must be first
		if filepath.Base(paths[0]) != "config.json" {
			t.Errorf("first path should be Docker config.json, got %q", paths[0])
		}
		// Podman runtime second
		if filepath.Base(paths[1]) != "auth.json" {
			t.Errorf("second path should be Podman auth.json, got %q", paths[1])
		}
		// Podman config third
		if filepath.Base(paths[2]) != "auth.json" {
			t.Errorf("third path should be Podman auth.json, got %q", paths[2])
		}
	})
}

func TestNewCredentialFunc(t *testing.T) {
	t.Run("no credential files returns anonymous func", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Point all paths to non-existent subdirectories within tmpDir.
		// ORAS NewStore succeeds with empty config for non-existent files,
		// so this tests anonymous credential return.
		t.Setenv(dockerConfigDirEnv, filepath.Join(tmpDir, "docker-absent"))
		t.Setenv(xdgRuntimeDirEnv, filepath.Join(tmpDir, "runtime-absent"))

		credFunc, err := NewCredentialFunc()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if credFunc == nil {
			t.Fatal("expected non-nil credential func")
		}

		cred, err := credFunc(context.Background(), "ghcr.io")
		if err != nil {
			t.Fatalf("unexpected error from credential func: %v", err)
		}
		if cred != auth.EmptyCredential {
			t.Errorf("expected empty credential, got %+v", cred)
		}
	})

	t.Run("Docker-only system returns credentials from Docker config", func(t *testing.T) {
		tmpDir := t.TempDir()
		dockerDir := filepath.Join(tmpDir, "docker")
		if err := os.MkdirAll(dockerDir, 0o700); err != nil {
			t.Fatal(err)
		}
		writeAuthFile(t, filepath.Join(dockerDir, "config.json"), "ghcr.io", "dockeruser", "dockerpass")
		t.Setenv(dockerConfigDirEnv, dockerDir)
		t.Setenv(xdgRuntimeDirEnv, "")

		credFunc, err := NewCredentialFunc()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cred, err := credFunc(context.Background(), "ghcr.io")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cred.Username != "dockeruser" {
			t.Errorf("username = %q, want %q", cred.Username, "dockeruser")
		}
		if cred.Password != "dockerpass" {
			t.Errorf("password = %q, want %q", cred.Password, "dockerpass")
		}
	})

	t.Run("Podman-only system returns credentials from Podman runtime", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Empty Docker config (no credentials)
		dockerDir := filepath.Join(tmpDir, "docker")
		if err := os.MkdirAll(dockerDir, 0o700); err != nil {
			t.Fatal(err)
		}
		writeAuthFile(t, filepath.Join(dockerDir, "config.json"), "", "", "")

		// Podman runtime with credentials
		runtimeDir := filepath.Join(tmpDir, "runtime")
		containersDir := filepath.Join(runtimeDir, "containers")
		if err := os.MkdirAll(containersDir, 0o700); err != nil {
			t.Fatal(err)
		}
		writeAuthFile(t, filepath.Join(containersDir, "auth.json"), "ghcr.io", "podmanuser", "podmanpass")

		t.Setenv(dockerConfigDirEnv, dockerDir)
		t.Setenv(xdgRuntimeDirEnv, runtimeDir)

		credFunc, err := NewCredentialFunc()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cred, err := credFunc(context.Background(), "ghcr.io")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cred.Username != "podmanuser" {
			t.Errorf("username = %q, want %q", cred.Username, "podmanuser")
		}
		if cred.Password != "podmanpass" {
			t.Errorf("password = %q, want %q", cred.Password, "podmanpass")
		}
	})

	t.Run("Podman config path returns credentials", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Empty Docker config (no credentials)
		dockerDir := filepath.Join(tmpDir, "docker")
		if err := os.MkdirAll(dockerDir, 0o700); err != nil {
			t.Fatal(err)
		}
		writeAuthFile(t, filepath.Join(dockerDir, "config.json"), "", "", "")

		// Podman config with credentials ($HOME/.config/containers/auth.json)
		homeDir := filepath.Join(tmpDir, "home")
		containersDir := filepath.Join(homeDir, ".config", "containers")
		if err := os.MkdirAll(containersDir, 0o700); err != nil {
			t.Fatal(err)
		}
		writeAuthFile(t, filepath.Join(containersDir, "auth.json"), "ghcr.io", "podmancfguser", "podmancfgpass")

		t.Setenv(dockerConfigDirEnv, dockerDir)
		t.Setenv(xdgRuntimeDirEnv, "")
		t.Setenv("HOME", homeDir)

		credFunc, err := NewCredentialFunc()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cred, err := credFunc(context.Background(), "ghcr.io")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cred.Username != "podmancfguser" {
			t.Errorf("username = %q, want %q", cred.Username, "podmancfguser")
		}
		if cred.Password != "podmancfgpass" {
			t.Errorf("password = %q, want %q", cred.Password, "podmancfgpass")
		}
	})

	t.Run("Docker credentials take priority over Podman", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Docker config with credentials
		dockerDir := filepath.Join(tmpDir, "docker")
		if err := os.MkdirAll(dockerDir, 0o700); err != nil {
			t.Fatal(err)
		}
		writeAuthFile(t, filepath.Join(dockerDir, "config.json"), "ghcr.io", "dockeruser", "dockerpass")

		// Podman runtime also with credentials for the same registry
		runtimeDir := filepath.Join(tmpDir, "runtime")
		containersDir := filepath.Join(runtimeDir, "containers")
		if err := os.MkdirAll(containersDir, 0o700); err != nil {
			t.Fatal(err)
		}
		writeAuthFile(t, filepath.Join(containersDir, "auth.json"), "ghcr.io", "podmanuser", "podmanpass")

		t.Setenv(dockerConfigDirEnv, dockerDir)
		t.Setenv(xdgRuntimeDirEnv, runtimeDir)

		credFunc, err := NewCredentialFunc()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cred, err := credFunc(context.Background(), "ghcr.io")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cred.Username != "dockeruser" {
			t.Errorf("username = %q, want %q (Docker should take priority)", cred.Username, "dockeruser")
		}
	})

	t.Run("missing Podman paths do not cause errors", func(t *testing.T) {
		tmpDir := t.TempDir()
		dockerDir := filepath.Join(tmpDir, "docker")
		if err := os.MkdirAll(dockerDir, 0o700); err != nil {
			t.Fatal(err)
		}
		writeAuthFile(t, filepath.Join(dockerDir, "config.json"), "ghcr.io", "dockeruser", "dockerpass")

		// XDG_RUNTIME_DIR points to a directory that does NOT have containers/auth.json
		t.Setenv(dockerConfigDirEnv, dockerDir)
		t.Setenv(xdgRuntimeDirEnv, filepath.Join(tmpDir, "no-such-runtime"))

		credFunc, err := NewCredentialFunc()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cred, err := credFunc(context.Background(), "ghcr.io")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cred.Username != "dockeruser" {
			t.Errorf("username = %q, want %q", cred.Username, "dockeruser")
		}
	})
}

// writeAuthFile creates a Docker/Podman-compatible auth config file with
// the given registry credentials. If registry is empty, writes an empty config.
func writeAuthFile(t *testing.T, path, registry, username, password string) {
	t.Helper()
	config := map[string]interface{}{
		"auths": map[string]interface{}{},
	}
	if registry != "" {
		config["auths"] = map[string]interface{}{
			registry: map[string]string{
				"username": username,
				"password": password,
			},
		}
	}
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("marshaling auth config: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("writing auth file %s: %v", path, err)
	}
}

func mustHomeDir(t *testing.T) string {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}
	return home
}
