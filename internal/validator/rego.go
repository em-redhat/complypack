// SPDX-License-Identifier: Apache-2.0

package validator

import (
	"github.com/open-policy-agent/opa/v1/ast"
)

// CheckRego validates Rego policy syntax and compilation.
// Returns a list of validation errors (empty if valid).
func CheckRego(filename string, src string) []error {
	mod, err := ast.ParseModuleWithOpts(filename, src, ast.ParserOptions{RegoVersion: ast.RegoV1})
	if err != nil {
		return []error{err}
	}

	compiler := ast.NewCompiler()
	compiler.Compile(map[string]*ast.Module{filename: mod})
	if compiler.Failed() {
		errs := make([]error, len(compiler.Errors))
		for i, e := range compiler.Errors {
			errs[i] = e
		}
		return errs
	}

	return nil
}
