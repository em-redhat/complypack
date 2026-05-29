// SPDX-License-Identifier: Apache-2.0

package complypack

import (
	"context"
	"errors"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content/memory"
)

func TestVerificationOptionsValidation(t *testing.T) {
	t.Run("both keyed and keyless verification", func(t *testing.T) {
		opts := &unpackOptions{
			verifyKeyPath:  "/key.pub",
			verifyCertPath: "/cert.pem",
			verifyIssuer:   "https://issuer.example.com",
			verifyIdentity: "user@example.com",
		}

		err := validateVerificationOptions(opts)
		if err == nil {
			t.Error("expected error for both verification options set")
		}
		if !errors.Is(err, ErrVerificationFailed) {
			t.Errorf("expected ErrVerificationFailed, got %v", err)
		}
	})

	t.Run("keyless without all fields", func(t *testing.T) {
		tests := []struct {
			name string
			opts *unpackOptions
		}{
			{
				name: "missing issuer",
				opts: &unpackOptions{
					verifyCertPath: "/cert.pem",
					verifyIdentity: "user@example.com",
				},
			},
			{
				name: "missing identity",
				opts: &unpackOptions{
					verifyCertPath: "/cert.pem",
					verifyIssuer:   "https://issuer.example.com",
				},
			},
			{
				name: "missing cert",
				opts: &unpackOptions{
					verifyIssuer:   "https://issuer.example.com",
					verifyIdentity: "user@example.com",
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := validateVerificationOptions(tt.opts)
				if err == nil {
					t.Error("expected error for incomplete keyless verification")
				}
			})
		}
	})

	t.Run("keyed verification only", func(t *testing.T) {
		opts := &unpackOptions{
			verifyKeyPath: "/key.pub",
		}

		err := validateVerificationOptions(opts)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("keyless verification complete", func(t *testing.T) {
		opts := &unpackOptions{
			verifyCertPath: "/cert.pem",
			verifyIssuer:   "https://issuer.example.com",
			verifyIdentity: "user@example.com",
		}

		err := validateVerificationOptions(opts)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("no verification", func(t *testing.T) {
		opts := &unpackOptions{}

		err := validateVerificationOptions(opts)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestVerifyStub(t *testing.T) {
	// This test verifies the stub returns "not implemented" error
	ctx := context.Background()
	store := memory.New()
	desc := ocispec.Descriptor{}
	opts := &unpackOptions{verifyKeyPath: "/key.pub"}

	err := verify(ctx, store, desc, opts)
	if err == nil {
		t.Error("expected not implemented error from stub")
	}
}
