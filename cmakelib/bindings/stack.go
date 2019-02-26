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

package bindings

type varStack struct {
	p *varStack
	v map[string]string
}

// Push constructs and returns a new child stack.
func (vs *varStack) Push() *varStack {
	return &varStack{vs, make(map[string]string)}
}

// Set sets the value associated with the key in the current scope.
func (vs *varStack) Set(key, value string) {
	vs.v[key] = value
}

// Get returns the associated value in the nearest scope or the empty string.
func (vs *varStack) Get(key string) string {
	val, ok := vs.v[key]
	if ok {
		return val
	}
	if vs.p != nil {
		return vs.p.Get(key)
	}
	return ""
}

// Values returns the map[string]string for the nested scopes.
func (vs *varStack) Values() map[string]string {
	values := make(map[string]string)
	for _, s := range vs.rootPath() {
		for key, val := range s.v {
			if val == "" {
				delete(values, key)
			} else {
				values[key] = val
			}
		}
	}
	return values
}

// rootPath returns the path from the current scope to the root as a slice.
func (vs *varStack) rootPath() []*varStack {
	path := []*varStack{vs}
	for p := vs.p; p != nil; p = p.p {
		path = append(path, p)
	}
	for left, right := 0, len(path)-1; left < right; left, right = left+1, right-1 {
		path[left], path[right] = path[right], path[left]
	}
	return path
}
