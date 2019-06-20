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

// Visitor is an interface which visit on the provided path.
type Visitor interface {
	Enter(dir Path) ([]Path, error) // Preorder, returns the paths of children to visit. Children must be relative to dir.
	Leave(dir Path) error           // Postorder, called after children are visited.
}

// PreVisitor is a single-function pre-order PathVisitor implementation.
type PreVisitor func(Path) ([]Path, error)

// Enter implements Visitor for PrePathVisitor.
func (p PreVisitor) Enter(dir Path) ([]Path, error) { return p(dir) }

// Leave implements Visitor for PrePathVisitor.
func (PreVisitor) Leave(Path) error { return nil }

// Walk traverses the directory at root in depth-first order, calling visit on
// selected subdirectories, begining at root.
func Walk(root Path, visit Visitor) error {
	children, err := visit.Enter(root)
	if err != nil {
		return err
	}
	for _, child := range children {
		if err := Walk(Join(root, child), visit); err != nil {
			return err
		}
	}
	return visit.Leave(root)
}
