// SPDX-License-Identifier: Apache-2.0

package complypack

import (
	"testing"
)

func TestPackOptions(t *testing.T) {
	t.Run("WithAnnotations", func(t *testing.T) {
		opts := &packOptions{}
		annotations := map[string]string{
			"key1": "value1",
			"key2": "value2",
		}
		opt := WithAnnotations(annotations)
		opt(opts)

		if len(opts.annotations) != 2 {
			t.Errorf("annotations count = %d, want 2", len(opts.annotations))
		}
		if opts.annotations["key1"] != "value1" {
			t.Errorf("annotations[key1] = %q, want %q", opts.annotations["key1"], "value1")
		}
	})

	t.Run("multiple annotations compose", func(t *testing.T) {
		opts := &packOptions{}
		WithAnnotations(map[string]string{"foo": "bar"})(opts)
		WithAnnotations(map[string]string{"baz": "qux"})(opts)

		if opts.annotations["foo"] != "bar" {
			t.Error("first annotations option not applied")
		}
		if opts.annotations["baz"] != "qux" {
			t.Error("second annotations option not applied")
		}
	})
}
