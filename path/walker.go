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

// Visitor is a interface which performs an action on the provided string-valued path.
type Visitor interface {
	Enter(dir string) ([]string, error) // Preorder, returns the paths of children to visit.
	Leave(dir string) error             // Postorder, called after children are visited.
}

// PathVisitor is an interface which visit on the provided path.
type PathVisitor interface {
	Enter(dir Path) ([]Path, error) // Preorder, returns the paths of children to visit.
	Leave(dir Path) error           // Postorder, called after children are visited.
}

// PreVisitor is a single-function pre-order Visitor implementation.
type PreVisitor func(string) ([]string, error)

// Enter implements Visitor for PreVisitor.
func (p PreVisitor) Enter(dir string) ([]string, error) { return p(dir) }

// Leave implements Visitor for PreVisitor.
func (PreVisitor) Leave(string) error { return nil }

// PrePathVisitor is a single-function pre-order PathVisitor implementation.
type PrePathVisitor func(Path) ([]Path, error)

// Enter implements Visitor for PrePathVisitor.
func (p PrePathVisitor) Enter(dir Path) ([]Path, error) { return p(dir) }

// Leave implements Visitor for PrePathVisitor.
func (PrePathVisitor) Leave(Path) error { return nil }

// wrapper is a PathVisitor wrapping a Visitor.
type wrapper struct {
	v Visitor
}

// AsPathVisitor wraps a Visitor as a PathVisitor by converting the relevant arguments and returns.
func AsPathVisitor(v Visitor) PathVisitor {
	return wrapper{v}
}

// Enter implements PathVisitor for Visitor.
func (w wrapper) Enter(dir Path) ([]Path, error) {
	cs, err := w.v.Enter(dir.String())
	return ToPaths(cs), err
}

// Leave implements PathVisitor for Visitor.
func (w wrapper) Leave(dir Path) error {
	return w.v.Leave(dir.String())
}

// Walk traverses the directory at root in depth-first order, calling visit on
// selected subdirectories, begining at root.
func Walk(root string, visit Visitor) error {
	return WalkPath(New(root), AsPathVisitor(visit))
}

// Walk traverses the directory at root in depth-first order, calling visit on
// selected subdirectories, begining at root.
func WalkPath(root Path, visit PathVisitor) error {
	children, err := visit.Enter(root)
	if err != nil {
		return err
	}
	for _, child := range children {
		if err := WalkPath(Join(root, child), visit); err != nil {
			return err
		}
	}
	return visit.Leave(root)
}
