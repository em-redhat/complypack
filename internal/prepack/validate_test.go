// SPDX-License-Identifier: Apache-2.0

package prepack

import (
	"context"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/complytime/complypack/internal/evaluator"
	"github.com/complytime/complypack/schemas"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadTestCUESchema(t *testing.T, platform string) cue.Value {
	t.Helper()
	data, err := schemas.GetBuiltInCUESchema(platform)
	require.NoError(t, err)

	ctx := cuecontext.New()
	val := ctx.CompileBytes(data)
	require.NoError(t, val.Err())
	return val
}

func TestValidate_ValidPolicies(t *testing.T) {
	ctx := context.Background()
	eval := &evaluator.OPA{}
	schema := loadTestCUESchema(t, "kubernetes")

	result, err := Validate(ctx, "testdata/valid", eval, schema, ValidationOptions{})
	require.NoError(t, err)

	assert.Equal(t, 1, result.FilesChecked)
	assert.Empty(t, result.SyntaxErrors)
	assert.Empty(t, result.ContractViolations)
	assert.True(t, result.Valid())
}

func TestValidate_SyntaxError(t *testing.T) {
	ctx := context.Background()
	eval := &evaluator.OPA{}
	schema := loadTestCUESchema(t, "kubernetes")

	result, err := Validate(ctx, "testdata/syntax-error", eval, schema, ValidationOptions{})
	require.NoError(t, err)

	assert.Equal(t, 1, result.FilesChecked)
	assert.NotEmpty(t, result.SyntaxErrors)
	assert.Empty(t, result.ContractViolations, "should skip contract check on syntax failure")
	assert.False(t, result.Valid())
}

func TestValidate_ContractViolation(t *testing.T) {
	ctx := context.Background()
	eval := &evaluator.OPA{}
	schema := loadTestCUESchema(t, "kubernetes")

	result, err := Validate(ctx, "testdata/contract-violation", eval, schema, ValidationOptions{})
	require.NoError(t, err)

	assert.Equal(t, 1, result.FilesChecked)
	assert.Empty(t, result.SyntaxErrors)
	assert.NotEmpty(t, result.ContractViolations)
	assert.False(t, result.Valid())
}

func TestValidate_EmptyDirectory(t *testing.T) {
	ctx := context.Background()
	eval := &evaluator.OPA{}
	schema := loadTestCUESchema(t, "kubernetes")

	result, err := Validate(ctx, "testdata/empty", eval, schema, ValidationOptions{})
	require.NoError(t, err)

	assert.Equal(t, 0, result.FilesChecked)
	assert.True(t, result.Valid())
}

func TestValidate_SkipTests(t *testing.T) {
	ctx := context.Background()
	eval := &evaluator.OPA{}
	schema := loadTestCUESchema(t, "kubernetes")

	result, err := Validate(ctx, "testdata/valid", eval, schema, ValidationOptions{
		SkipTests: true,
	})
	require.NoError(t, err)

	assert.True(t, result.TestsSkipped)
	assert.Nil(t, result.TestResults)
	assert.True(t, result.Valid())
}

func TestValidate_NoSchema(t *testing.T) {
	ctx := context.Background()
	eval := &evaluator.OPA{}

	result, err := Validate(ctx, "testdata/valid", eval, cue.Value{}, ValidationOptions{})
	require.NoError(t, err)

	assert.Equal(t, 1, result.FilesChecked)
	assert.Empty(t, result.SyntaxErrors)
	assert.Empty(t, result.ContractViolations, "contract check should be skipped without schema")
	assert.True(t, result.Valid())
}

func TestValidate_NonexistentDirectory(t *testing.T) {
	ctx := context.Background()
	eval := &evaluator.OPA{}

	_, err := Validate(ctx, "testdata/nonexistent", eval, cue.Value{}, ValidationOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "collecting policy files")
}

func TestCollectFiles(t *testing.T) {
	files, err := collectFiles("testdata/valid", ".rego")
	require.NoError(t, err)
	assert.Len(t, files, 1)

	files, err = collectFiles("testdata/empty", ".rego")
	require.NoError(t, err)
	assert.Len(t, files, 0)
}

func TestIsTestFile(t *testing.T) {
	assert.True(t, isTestFile("policy_test.rego", ".rego"))
	assert.False(t, isTestFile("policy.rego", ".rego"))
	assert.False(t, isTestFile("test_helper.rego", ".rego"))
}
