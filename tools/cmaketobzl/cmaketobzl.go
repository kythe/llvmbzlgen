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
	"github.com/kythe/llvmbzlgen/writer"
)

type blockCounter struct {
	begin string
	end   string
	count int
}

func NewCounter(begin string) *blockCounter {
	return &blockCounter{begin, "end" + begin, 0}
}

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

type options struct {
	macroName   string
	shouldPrint Predicate
	shouldAdd   Predicate
	excludePath Predicate
}

type eval struct {
	p *ast.Parser
	o options

	w    *writer.StarlarkWriter
	v    *bindings.Mapping
	path []string
}

type Option func(*eval)
type Predicate func(string) bool

func PrintCommands(p Predicate) Option {
	return func(e *eval) {
		e.o.shouldPrint = p
	}
}

func RecurseCommands(p Predicate) Option {
	return func(e *eval) {
		e.o.shouldAdd = p
	}
}

func ExcludePaths(p Predicate) Option {
	return func(e *eval) {
		e.o.excludePath = p
	}
}

func DefineVars(vars map[string]string) Option {
	return func(e *eval) {
		for k, v := range vars {
			e.v.Set(k, v)
		}
	}
}

func Matching(pat string) Predicate {
	return regexp.MustCompile(pat).MatchString
}

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

func (e *eval) parse(input io.Reader) (*ast.CMakeFile, error) {
	return e.p.Parse(input)
}

func (e *eval) parseFile(path string) (*ast.CMakeFile, error) {
	if input, err := os.Open(path); err != nil {
		return nil, err
	} else {
		return e.parse(input)
	}
}

func (e *eval) walk(paths []string) error {
	if err := e.w.BeginMacro(e.o.macroName); err != nil {
		return err
	}
	root, paths := SplitCommonRoot(paths)
	e.path = append(e.path, root)
	for _, p := range paths {
		if err := e.AddSubdirectory(p); err != nil {
			return err
		}
	}
	return e.w.EndMacro()
}

type dispatchFn func(*commandList) (dispatchFn, error)

type commandList []ast.CommandInvocation

func (l *commandList) Advance() bool {
	if len(*l) > 0 {
		*l = (*l)[1:]
		return len(*l) > 0
	}
	return false
}

func (l *commandList) Head() *ast.CommandInvocation {
	if len(*l) > 0 {
		return &(*l)[0]
	}
	return nil
}

func (e *eval) shouldPrint(name string) bool {
	if e.o.shouldPrint != nil {
		return e.o.shouldPrint(name)
	}
	return false

}

func (e *eval) shouldAdd(name string) bool {
	if e.o.shouldAdd != nil {
		return e.o.shouldAdd(name)
	}
	return false
}

func (e *eval) excludePath(dirpath string) bool {
	if e.o.excludePath != nil {
		return e.o.excludePath(dirpath)
	}
	return false
}

func (e *eval) dispatch(cmds *commandList) (dispatchFn, error) {
	name := string(cmds.Head().Name)
	if e.shouldPrint(name) {
		e.PrintCommand(cmds.Head())
	}

	switch name {
	// TODO(shahms): Actually process these.
	case "if", "function", "foreach", "macro":
		counter := NewCounter(name)
		for counter.Count(name) && cmds.Advance() {
			name = string(cmds.Head().Name)
		}
		return e.dispatch, nil
	// TODO(shahms): Support setting values.
	case "set":
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

func (e *eval) AddSubdirectory(dirpath string) error {
	if err := e.enterDirectory(dirpath); err != nil {
		return err
	}
	file, err := e.parseFile(path.Join(path.Join(e.path...), "CMakeLists.txt"))
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

func (e *eval) CurrentDirectory() string {
	return "/" + path.Join(e.path[1:]...)
}

func (e *eval) enterDirectory(dirpath string) error {
	if err := e.w.PushDirectory(dirpath); err != nil {
		return err
	}
	e.v.Push()
	e.path = append(e.path, dirpath)
	e.v.Set("CMAKE_CURRENT_SOURCE_DIR", e.CurrentDirectory())
	e.v.Set("CMAKE_CURRENT_BINARY_DIR", e.CurrentDirectory())
	return nil
}

func (e *eval) exitDirectory(path string) error {
	e.v.Pop()
	e.path = e.path[:len(e.path)-1]
	tail, err := e.w.PopDirectory()
	if tail != path {
		return fmt.Errorf("unexpected directory state %v != %v", tail, path)
	}
	return err
}

func (e *eval) PrintCommand(command *ast.CommandInvocation) error {
	return e.w.WriteCommand(string(command.Name), command.Arguments.Eval(e.v)...)
}

type Path []string

func NewPath(s string) Path {
	p := strings.Split(path.Clean(s), "/")
	if p[0] == "" {
		p[0] = "/"
	}
	return p
}

func (p Path) LessThan(o Path) bool {
	switch {
	case len(p) < len(o):
		return true
	case len(p) > len(o):
		return false
	}
	for i := range p {
		if p[i] != o[i] {
			return p[i] < o[i]
		}
	}
	return false
}

func (p Path) String() string {
	return path.Join([]string(p)...)
}

func SplitCommonRoot(paths []string) (string, []string) {
	var split []Path
	for _, p := range paths {
		split = append(split, NewPath(p))
	}
	root := LongestCommonPrefix(split)
	for i, p := range split {
		paths[i] = p[len(root):].String()
	}
	return root.String(), paths
}

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

func main() {
	flag.Parse()
	eval := NewEvaluator(os.Stdout,
		DefineVars(map[string]string{
			"LLVM_MAIN_INCLUDE_DIR": "/include",
			"LLVM_INCLUDE_DIR":      "/include",
			"CLANG_SOURCE_DIR":      "/tools/clang",
			"CLANG_BINARY_DIR":      "/tools/clang",
		}),
		ExcludePaths(Matching(`(^|/)(unittests|examples|cmake)($|/)`)),
		RecurseCommands(Matching(`add(_\w+)?_subdirectory`)),
		PrintCommands(Matching("^("+strings.Join([]string{
			"configure_file", "set",
			"add_llvm_library", "add_clang_library", "add_llvm_target",
			"add_tablegen", "tablegen", "clang_diag_gen", "clang_tablegen", "add_public_tablegen_target",
		}, "|")+")$")))
	if err := eval.walk(flag.Args()); err != nil {
		log.Fatal(err)
	}
}
