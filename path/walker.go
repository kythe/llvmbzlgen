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

// Type Visitor is a function which performs an action on the provided path.
// The return value is a slice of subdirectories to visit, a function to call to "close"
// the visited directory on successful visitation or an error.
type Visitor func(string) ([]string, func() error, error)

// Type PathVisitor is a function which performs an action on the provided path.
// The return value is a slice of subdirectories to visit, a function to call to "close"
// the visited directory on successful visitation or an error.
type PathVisitor func(Path) ([]Path, func() error, error)

// Walk traverses the directory at root in depth-first order, calling visit on
// selected subdirectories, begining at root.
func Walk(root string, visit Visitor) error {
	return WalkPath(New(root), func(path Path) ([]Path, func() error, error) {
		children, close, err := visit(path.String())
		return ToPaths(children), close, err
	})
}

// Walk traverses the directory at root in depth-first order, calling visit on
// selected subdirectories, begining at root.
func WalkPath(root Path, visit PathVisitor) error {
	children, close, err := visit(root)
	if err != nil {
		return err
	}
	for _, child := range children {
		if err := WalkPath(root.Join(child), visit); err != nil {
			return err
		}
	}
	if close != nil {
		return close()
	}
	return nil
}
