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

package path

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestStripCommonRoot(t *testing.T) {
	type result struct {
		Root  string
		Paths []string
	}
	type test struct {
		paths    []string
		expected result
	}
	newResult := func(root string, paths []string) result {
		return result{root, paths}
	}
	tests := []test{
		// Simple common root.
		{[]string{"/a/b/c", "/a/b/d", "/a/b/"}, result{"/a/b", []string{"c", "d", ""}}},
		// Only whole-segments allowed.
		{[]string{"a/b/c", "a/bb/c"}, result{"a", []string{"b/c", "bb/c"}}},
		// No common root.
		{[]string{"a/b/c", "c/d/e", "d/e/f"}, result{"", []string{"a/b/c", "c/d/e", "d/e/f"}}},
	}

	for _, tc := range tests {
		r := newResult(SplitCommonRootString(tc.paths))
		if diff := cmp.Diff(r, tc.expected); diff != "" {
			t.Errorf("Unexpected root %#v:\n%s", tc.paths, diff)
		}
	}
}

func TestLongestCommonPrefix(t *testing.T) {
	type test struct {
		input    []string
		expected string
	}
	tests := []test{
		{[]string{"a/b/c", "a/b", "a/c/b"}, "a"},
		{[]string{"/a/b/c", "/a/b", "/a/c/b"}, "/a"},
		{[]string{"a/bb/c", "a/b", "a/b/c"}, "a"},
		{[]string{"a/c", "a/b", "b/c"}, ""},
	}

	for _, tc := range tests {
		r := LongestCommonPrefix(ToPaths(tc.input)).String()
		if diff := cmp.Diff(r, tc.expected); diff != "" {
			t.Errorf("Unexpected prefix %#v:\n%s", tc.input, diff)
		}
	}
}

func TestPathJoin(t *testing.T) {
	root := New("/a/b/c")
	children := []Path{New("d"), New("e"), New("f")}
	expected := New("/a/b/c/d/e/f")
	if diff := cmp.Diff(Join(root, children...), expected); diff != "" {
		t.Errorf("Unexpected path:\n%s", diff)
	}

	root = Append(root, children...)
	if diff := cmp.Diff(root, expected); diff != "" {
		t.Errorf("Unexpected path:\n%s", diff)
	}
}

func TestAbsolutePathJoin(t *testing.T) {
	root := New("a")
	tests := []struct {
		value    []Path
		expected Path
	}{
		{[]Path{New("/a")}, New("a/a")},
		{[]Path{New("b"), New("/c")}, New("a/b/c")},
	}
	for _, test := range tests {
		if diff := cmp.Diff(Join(root, test.value...), test.expected); diff != "" {
			t.Errorf("Unexpected path:\n%s", diff)
		}
	}
}

func TestWalkPath(t *testing.T) {
	walk := New("/a/b/c/d/e/f/g")
	var found []string
	expected := []string{
		"/",
		"/a",
		"/a/b",
		"/a/b/c",
		"/a/b/c/d",
		"/a/b/c/d/e",
		"/a/b/c/d/e/f",
		"/a/b/c/d/e/f/g",
	}
	Walk(New("/"), PreVisitor(func(p Path) ([]Path, error) {
		found = append(found, p.String())
		if len(p) == len(walk) {
			return nil, nil
		}
		return []Path{walk[len(p) : len(p)+1]}, nil
	}))
	if diff := cmp.Diff(found, expected); diff != "" {
		t.Errorf("Unexpected traversal:\n%s", diff)
	}
}

func TestPathLen(t *testing.T) {
	type test struct {
		input    string
		expected int
	}
	tests := []test{
		{"/", 1},
		{"/a/b/c", 4},
		{"a/b/c", 3},
	}
	for _, test := range tests {
		path := New(test.input)
		if len(path) != test.expected {
			t.Errorf("len(%s) = %d; want %d (%#v)", path, len(path), test.expected, path)
		}
	}
}
