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

// Package bindings implements CMake-style variable bindings.
package bindings

// Mapping is a stack of map[string]string for CMake variables.
type Mapping struct {
	vs *varStack
}

// New returns a new, empty, variable stack.
func New() *Mapping {
	m := &Mapping{}
	m.Push()
	return m
}

// Push pushes a new variable binding scope.
func (m *Mapping) Push() {
	m.vs = m.vs.Push()
}

// Pop removes the most recently pushed scope.
func (m *Mapping) Pop() {
	m.vs = m.vs.p
}

// Set sets a key to a particular value in the current scope.
func (m *Mapping) Set(key, value string) {
	m.vs.Set(key, value)
}

// SetParent sets a key to a particular value in the parent scope.
func (m *Mapping) SetParent(key, value string) {
	m.vs.p.Set(key, value)
}

// Get looks from the current scope up to find the nearest value for key.
func (m *Mapping) Get(key string) string {
	return m.vs.Get(key)
}

// GetCache returns the associated value from the variable cache (not implemented).
func (m *Mapping) GetCache(key string) string {
	return ""
}

// GetEnv returns the corresponding environment variable or the empty string (not implemented).
func (m *Mapping) GetEnv(key string) string {
	return ""
}

// Values returns the currently set values as a map[string]string.
func (m *Mapping) Values() map[string]string {
	return m.vs.Values()
}
