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

// Package path implements path manipulation routines used by cmaketobzl.
package path

import (
	"path/filepath"
	"strings"
)

// Path is a slice of string segments, representing a filesystem path.
type Path []string

// Split cleans and splits the system-delimited filesystem path.
func New(s string) Path {
	s = filepath.ToSlash(filepath.Clean(s))
	switch {
	case len(s) == 0:
		return nil
	case s == "/":
		return Path{"/"}
	case s[0] == '/':
		return append(Path{"/"}, strings.Split(s[1:], "/")...)
	default:
		return strings.Split(s, "/")
	}
}

// ToPaths cleans and splits each of the system-delimited filesystem paths.
func ToPaths(paths []string) []Path {
	split := make([]Path, len(paths))
	for i, p := range paths {
		split[i] = New(p)
	}
	return split
}

// LessThan provides lexicographic ordering of Paths.
func (p Path) LessThan(o Path) bool {
	for i := 0; ; i++ {
		if i >= len(p) {
			return i < len(o)
		} else if i >= len(o) {
			return false
		} else if p[i] != o[i] {
			return p[i] < o[i]
		}
	}
	return false
}

// String returns the properly platform-delimited form of the path.
func (p Path) String() string {
	return filepath.Join([]string(p)...)
}

// Append appends additional elements to the end of path.
func (p *Path) Append(elem ...Path) {
	for _, e := range elem {
		*p = append(*p, e...)
	}
}

// AppendString appends additional string elements to the end of path.
func (p *Path) AppendString(elem ...string) {
	p.Append(ToPaths(elem)...)
}

// Join joins path and any number of additional elements, returning the result.
func (p Path) Join(elem ...Path) Path {
	root := p[:]
	root.Append(elem...)
	return root
}

// JoinString joins path and any number of additional string elements, returning the result.
func (p Path) JoinString(elem ...string) Path {
	root := p[:]
	root.Append(ToPaths(elem)...)
	return root
}

// Join joins any number of paths and returns the result.
func Join(elem ...Path) Path {
	if len(elem) == 0 {
		return nil
	}
	return elem[0].Join(elem[1:]...)
}

// SplitCommonRoot finds the longest command whole-segment prefix of the provided
// path and returns that along with each path stripped of that prefix.
func SplitCommonRoot(paths []Path) (Path, []Path) {
	root := LongestCommonPrefix(paths)
	if len(root) == 0 {
		return root, paths
	}
	result := make([]Path, len(paths))
	for i, p := range paths {
		result[i] = p[len(root):]
	}
	return root, result
}

// SplitCommonRootString finds the longest command whole-segment prefix of the provided
// path and returns that along with each path stripped of that prefix as /-delimited strings.
func SplitCommonRootString(paths []string) (string, []string) {
	root, stripped := SplitCommonRoot(ToPaths(paths))
	strings := make([]string, len(stripped))
	for i, p := range stripped {
		strings[i] = p.String()
	}
	return root.String(), strings
}

// LongestCommonPrefix returns the longest shared Path prefix of all of the paths.
func LongestCommonPrefix(paths []Path) Path {
	switch len(paths) {
	case 0:
		return nil
	case 1:
		return paths[0]
	}
	min, max := paths[0], paths[0]
	for _, p := range paths[1:] {
		switch {
		case p.LessThan(min):
			min = p
		case max.LessThan(p):
			max = p
		}
	}

	for i := 0; i < len(min) && i < len(max); i++ {
		if min[i] != max[i] {
			return min[:i]
		}
	}
	return min
}
