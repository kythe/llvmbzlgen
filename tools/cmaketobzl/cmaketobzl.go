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

package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/kythe/llvmbzlgen/cmakelib/ast"
	"github.com/kythe/llvmbzlgen/cmakelib/bindings"
	bzlpath "github.com/kythe/llvmbzlgen/path"
	"github.com/kythe/llvmbzlgen/writer"
)

// blockCounter counts active blocks of the given name for matching
// paired CMake commands.
type blockCounter struct {
	begin string // The beginning command, e.g. "if"
	end   string // The ending command, e.g. "endif"
	count int
}

// newCounter returns a new blockCounter instance which counts
// blocks delimited by begin and "end" + begin.
func newCounter(begin string) *blockCounter {
	return &blockCounter{begin, "end" + begin, 0}
}

// Count increments the internal counter if text matches the begin delimiter,
// decrements it if it matches the end delimiter and returns true if text
// matched a delimiter or the current count is greater than zero.
func (bc *blockCounter) Count(text string) bool {
	matched := true
	if text == bc.begin {
		bc.count += 1
	} else if text == bc.end {
		bc.count -= 1
	} else {
		matched = false
	}
	return matched || bc.count > 0
}

type eval struct {
	p *ast.Parser
	o options

	w    *writer.StarlarkWriter
	v    *bindings.Mapping
	root bzlpath.Path
	path bzlpath.Path
}

type options struct {
	macroName   string
	shouldPrint func(string) bool
	shouldAdd   func(string) bool
	excludePath func(string) bool
}

// Option is a configuration option for the CMake evaluator.
type Option func(*eval)

// PrintCommands configures the evaluator to print commands on the StarlarkWriter for which the supplied predicate returns true.
func PrintCommands(p func(string) bool) Option {
	return func(e *eval) { e.o.shouldPrint = p }
}

// RecurseCommands configures the evaluator to recurse into the subdirectory
// specified by the first argument to the command when the provided predicate returns true.
// By default only "add_subdirectory" is handled this way.
func RecurseCommands(p func(string) bool) Option {
	return func(e *eval) { e.o.shouldAdd = p }
}

// ExcludePaths configures the evaluator to omit particular paths entirely during traversal.
func ExcludePaths(p func(string) bool) Option {
	return func(e *eval) { e.o.excludePath = p }
}

// DefineVars configures the evaluator to predefine the specified variables.
func DefineVars(vars map[string]string) Option {
	return func(e *eval) {
		for k, v := range vars {
			e.v.Set(k, v)
		}
	}
}

// Matching compiles the provided pattern and returns a predicate for matching strings.
func Matching(pat string) func(string) bool {
	return regexp.MustCompile(pat).MatchString
}

// NewEvaluator returns a new CMake evaluator instance.
func NewEvaluator(w io.Writer, opts ...Option) *eval {
	e := &eval{
		p: ast.NewParser(),
		w: writer.NewStarlarkWriter(w),
		v: bindings.New(),
		o: options{
			macroName: "generated_cmake_targets",
			shouldAdd: func(n string) bool { return n == "add_subdirectory" },
		},
	}
	for _, o := range opts {
		o(e)
	}
	return e
}

// parse parses the provided input into a CMakeFile AST.
func (e *eval) parse(input io.Reader) (*ast.CMakeFile, error) {
	return e.p.Parse(input)
}

// parse parses the provided path into a CMakeFile AST.
func (e *eval) parseFile(path string) (*ast.CMakeFile, error) {
	input, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer input.Close()
	return e.parse(input)
}

// walk evaluates all of the provided CMakeLists.txt files into the body of a single Starlark macro..
func (e *eval) walk(paths []bzlpath.Path) error {
	if err := e.w.BeginMacro(e.o.macroName); err != nil {
		return err
	}
	root, paths := bzlpath.SplitCommonRoot(paths)
	e.root = root
	for _, p := range paths {
		if err := e.AddSubdirectory(p.String()); err != nil {
			return err
		}
	}
	return e.w.EndMacro()
}

// dispatchFunc is a function which handles the current command, updates the
// remaining list of commands and returns a dispatchFunc suitable for processing that remainder.
type dispatchFunc func(*commandList) (dispatchFunc, error)

// commandList is a slice of CMake CommandInvocation elements used for dispatch.
type commandList []ast.CommandInvocation

// Advance removes the top command from the list and returns true if there are more to process.
func (l *commandList) Advance() bool {
	if len(*l) > 0 {
		*l = (*l)[1:]
		return len(*l) > 0
	}
	return false
}

// Head returns the first command in the list, if any.
func (l *commandList) Head() *ast.CommandInvocation {
	if len(*l) > 0 {
		return &(*l)[0]
	}
	return nil
}

// shouldPrint returns true if the command given by name should be included in the Starlark output.
func (e *eval) shouldPrint(name string) bool {
	return e.o.shouldPrint != nil && e.o.shouldPrint(name)
}

// shouldAdd retruns true if the command given by name should be recursed into.
func (e *eval) shouldAdd(name string) bool {
	return e.o.shouldAdd != nil && e.o.shouldAdd(name)
}

// excludePath returns true if the path given by dirpath should be skipped.
func (e *eval) excludePath(dirpath string) bool {
	return e.o.excludePath != nil && e.o.excludePath(dirpath)
}

// dispatch evaluates the next command from cmds and returns a new dispatchFunc for handling the remainder.
func (e *eval) dispatch(cmds *commandList) (dispatchFunc, error) {
	name := string(cmds.Head().Name)
	if e.shouldPrint(name) {
		e.PrintCommand(cmds.Head())
	}

	switch name {
	// TODO(shahms): Actually process these.
	case "if", "function", "foreach", "macro":
		counter := newCounter(name)
		for counter.Count(name) && cmds.Advance() {
			name = string(cmds.Head().Name)
		}
		return e.dispatch, nil
	case "set":
		e.setVariable(cmds.Head().Arguments.Eval(e.v))
	case "unset":
		e.unsetVariable(cmds.Head().Arguments.Eval(e.v))
	}

	if e.shouldAdd(name) {
		args := cmds.Head().Arguments.Eval(e.v)
		if len(args) != 1 {
			return nil, fmt.Errorf("invalid number of arguments to directory command %s", cmds.Head().Pos)
		}
		if !e.excludePath(args[0]) {
			if err := e.AddSubdirectory(cmds.Head().Arguments.Eval(e.v)[0]); err != nil {
				return nil, err
			}
		}
	}
	cmds.Advance()
	return e.dispatch, nil
}

// setVariable sets the value of the variable designated by the remained, following the rules of
// https://cmake.org/cmake/help/v3.3/command/set.html#command:set
func (e *eval) setVariable(args []string) {
	if len(args) == 0 {
		log.Println("Cannot set a variable without a name")
		return
	}
	key, args := args[0], args[1:len(args)]
	switch {
	case len(args) > 0 && args[len(args)-1] == "PARENT_SCOPE":
		e.v.SetParent(key, strings.Join(args[0:len(args)-1], ";"))
	case len(args) >= 3 && args[len(args)-3] == "CACHE":
		e.v.SetCache(key, strings.Join(args[:len(args)-3], ";"))
	case len(args) >= 4 && args[len(args)-4] == "CACHE": // FORCE
		e.v.SetCache(key, strings.Join(args[:len(args)-4], ";"))
	default:
		e.v.Set(key, strings.Join(args, ";"))
	}
}

// unsetVariable unsets the value of the variable designated by the remained, following the rules of
// https://cmake.org/cmake/help/v3.3/command/set.html#command:unset
func (e *eval) unsetVariable(args []string) {
	switch {
	case len(args) == 0:
		log.Println("Cannot unset a variable without a name")
	case len(args) == 1:
		e.v.Set(args[0], "")
	case len(args) == 2 && args[1] == "PARENT_SCOPE":
		e.v.SetParent(args[0], "")
	case len(args) == 2 && args[1] == "CACHE":
		e.v.SetCache(args[0], "")
	default:
		log.Println("Ignoring invalid unset command")
	}
}

// AddSubdirectory recurses into the directory specified by dirpath and evaluates the CMakeLists.txt contained therein.
func (e *eval) AddSubdirectory(dirpath string) error {
	if err := e.enterDirectory(dirpath); err != nil {
		return err
	}
	file, err := e.parseFile(path.Join(e.root.String(), e.path.String(), "CMakeLists.txt"))
	if err != nil {
		return err
	}

	cmds := commandList(file.Commands)
	dispatch := e.dispatch
	for {
		dispatch, err := dispatch(&cmds)
		if err != nil {
			return err
		}
		if len(cmds) == 0 || dispatch == nil {
			break
		}

	}
	return e.exitDirectory(dirpath)
}

// ProjectRoot returns the path prefix for forming project-rooted absolute paths.
func (e *eval) ProjectRoot() string {
	// Use a fixed prefix so that paths formed by simple string concatenation don't
	// start with '//' which is often treated specially.
	return "/root"
}

// CurrentDirectory returns the relative, project-rooted path currently being traversed.
func (e *eval) CurrentDirectory() string {
	return path.Join(e.path...)
}

// enterDirectory pushes a new directory onto the stack, setting up necessary state, etc.
func (e *eval) enterDirectory(dirpath string) error {
	if err := e.w.PushDirectory(dirpath); err != nil {
		return err
	}
	e.v.Push()
	e.path = append(e.path, dirpath)
	e.v.Set("CMAKE_CURRENT_SOURCE_DIR", path.Join(e.ProjectRoot(), e.CurrentDirectory()))
	e.v.Set("CMAKE_CURRENT_BINARY_DIR", path.Join(e.ProjectRoot(), e.CurrentDirectory()))
	return nil
}

// exitDirectory pops the most recently entered directory off the stack.
func (e *eval) exitDirectory(path string) error {
	e.v.Pop()
	e.path = e.path[:len(e.path)-1]
	tail, err := e.w.PopDirectory()
	if tail != path {
		return fmt.Errorf("unexpected directory state %v != %v", tail, path)
	}
	return err
}

// PrintCommand writes the given command to the configured StarlarkWriter.
func (e *eval) PrintCommand(command *ast.CommandInvocation) error {
	return e.w.WriteCommand(string(command.Name), writer.ArgumentLiterals(command.Arguments.Eval(e.v)))
}

func main() {
	flag.Parse()
	eval := NewEvaluator(os.Stdout,
		ExcludePaths(Matching(`(^|/)(unittests|examples|cmake)($|/)`)),
		RecurseCommands(Matching(`add(_\w+)?_subdirectory`)),
		PrintCommands(Matching("^("+strings.Join([]string{
			"configure_file", "set",
			"add_llvm_library", "add_clang_library", "add_llvm_target",
			"add_tablegen", "tablegen", "clang_diag_gen", "clang_tablegen", "add_public_tablegen_target",
		}, "|")+")$")))
	if err := eval.walk(bzlpath.ToPaths(flag.Args())); err != nil {
		log.Fatal(err)
	}
}
