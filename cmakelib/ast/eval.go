/*
 * Copyright 2019 The Kythe Authors. All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package ast

import (
	"fmt"
	"strings"
)

// Eval uses the provided bindings to resolve any variable references and returns a slice
// corresponding to the argument values.
func (a *ArgumentList) Eval(vars Bindings) []string {
	var values []string
	for _, arg := range a.Values {
		values = append(values, arg.Eval(vars)...)
	}
	return values
}

// Eval returns a slice of argument values after resolving variable references from vars.
func (a *Argument) Eval(vars Bindings) []string {
	switch {
	case a.QuotedArgument != nil:
		return a.QuotedArgument.Eval(vars)
	case a.UnquotedArgument != nil:
		return a.UnquotedArgument.Eval(vars)
	case a.BracketArgument != nil:
		return a.BracketArgument.Eval(vars)
	case a.ArgumentList != nil:
		// Include the parens, but only for nested argument lists.
		values := []string{"("}
		values = append(values, a.ArgumentList.Eval(vars)...)
		return append(values, ")")
	}
	panic("Missing concrete argument!")
}

// Eval returns a slice of argument values after resolving variable references from vars.
// Semi-colon delimited lists are not separated.
func (a *QuotedArgument) Eval(vars Bindings) []string {
	var parts []string
	for _, e := range a.Elements {
		parts = append(parts, e.Eval(vars)...)
	}
	return []string{strings.Join(parts, "")}
}

// Eval returns a slice of values after resolving variable references using vars.
func (e *QuotedElement) Eval(vars Bindings) []string {
	if e.Ref != nil {
		return e.Ref.Eval(vars)
	}
	// TODO(shahms): Deal with escape sequences.
	return []string{e.Text}
}

// Eval returns a slice of argument values after resolving variable references from vars.
// Semi-colon delimited lists are separated.
func (a *UnquotedArgument) Eval(vars Bindings) []string {
	var parts []string
	for _, e := range a.Elements {
		parts = append(parts, e.Eval(vars)...)
	}
	return []string{strings.Join(parts, "")}
}

// Eval returns a slice of values after evaluating escape sequences
// and splitting on semicolons.
func (e *UnquotedElement) Eval(vars Bindings) []string {
	if e.Ref != nil {
		return e.Ref.Eval(vars)
	}
	// TODO(shahms): Deal with escape sequences and lists.
	return []string{e.Text}
}

// Eval returns a slice of values for the text of the argument.
func (a *BracketArgument) Eval(vars Bindings) []string {
	return []string{a.Text}
}

// Eval recursively resolved variable references using vars and returns the result.
func (v *VariableReference) Eval(vars Bindings) []string {
	var name []string
	for _, e := range v.Elements {
		name = append(name, e.Eval(vars)...)
	}
	var get func(string) string
	switch v.Domain {
	case DomainDefault:
		get = vars.Get
	case DomainCache:
		get = vars.GetCache
	case DomainEnv:
		get = vars.GetEnv
	case DomainMake:
		fallthrough
	default:
		panic(fmt.Sprintf("unrecognized domain: %#v", v.Domain))
	}
	return []string{get(strings.Join(name, ""))}
}

// Eval recursively resolved variable references using vars and returns the result.
func (v *VariableElement) Eval(vars Bindings) []string {
	parts := []string{v.Text}
	if v.Ref != nil {
		for _, p := range v.Ref.Eval(vars) {
			parts = append(parts, p)
		}
	}
	return []string{strings.Join(parts, "")}
}
