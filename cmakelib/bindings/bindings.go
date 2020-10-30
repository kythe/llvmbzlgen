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
// https://cmake.org/cmake/help/latest/manual/cmake-language.7.html#variables
package bindings

import "log"

// Mapping is a stack of map[string]string for CMake variables.
type Mapping struct {
	vs    []map[string]string
	cache map[string]string
}

// New returns a new, empty, variable stack.
func New() *Mapping {
	m := &Mapping{cache: make(map[string]string)}
	m.Push()
	return m
}

// Push pushes a new variable binding scope.
func (m *Mapping) Push() {
	m.vs = append(m.vs, make(map[string]string))
}

// Pop removes the most recently pushed scope.
func (m *Mapping) Pop() {
	m.vs = m.vs[0 : len(m.vs)-1]
}

// Depth returns the current mapping depth starting from 0.
func (m *Mapping) Depth() int {
	return len(m.vs) - 1
}

// Set sets a key to a particular value in the current scope.
// Setting a key to the empty string is equivalent to deleting it, in accordance with CMake semantics.
func (m *Mapping) Set(key, value string) {
	// Keep empty strings in the current scope as a tombstone to prevent searching in parent scopes.
	m.vs[len(m.vs)-1][key] = value
}

// SetParent sets a key to a particular value in the parent scope.
// Setting a key to the empty string is equivalent to deleting it, in accordance with CMake semantics.
func (m *Mapping) SetParent(key, value string) {
	if m.Depth() == 0 {
		log.Println("Attempt to set ", key, "in PARENT_SCOPE at root")
	} else {
		m.vs[len(m.vs)-2][key] = value
	}
}

// SetCache sets a key to a particular value in CACHE scope.
// Setting a key to the empty string is equivalent to deleting it, in accordance with CMake semantics.
func (m *Mapping) SetCache(key, value string) {
	m.cache[key] = value
}

// Get looks from the current scope up to find the nearest value for key.
// If they key is absent, returns the empty string.
// This matches the semantics of CMake variable lookup.
func (m *Mapping) Get(key string) string {
	for i := len(m.vs) - 1; i >= 0; i-- {
		val, ok := m.vs[i][key]
		if ok {
			return val
		}
	}
	// From https://cmake.org/cmake/help/latest/manual/cmake-language.7.html#variables
	// Variable references are looked up in the cache if not present in the current scope.
	return m.GetCache(key)
}

// GetCache returns the associated value from the variable cache or an empty string if not found.
func (m *Mapping) GetCache(key string) string {
	val, ok := m.cache[key]
	if ok {
		return val
	}
	return ""
}

// GetEnv returns the corresponding environment variable or the empty string (not implemented).
func (m *Mapping) GetEnv(key string) string {
	return ""
}

// Values returns the currently set values as a map[string]string.
// Keys set to the empty string will be omitted from the final map.
func (m *Mapping) Values() map[string]string {
	vals := make(map[string]string)
	for _, v := range m.vs {
		for key, val := range v {
			if val == "" {
				delete(vals, key)
			} else {
				vals[key] = val
			}
		}
	}
	return vals
}
