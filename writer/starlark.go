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

package writer

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

// StarlarkWriter is a simple type for writing basic Starlark macros with a consistent form.
type StarlarkWriter struct {
	w            *bufio.Writer
	buf          []string
	hasBody      bool
	currentMacro string
	dirStack     []string
}

// NewStarlarkWriter creates a new StarlarkWriter writing to the provided output.
func NewStarlarkWriter(w io.Writer) *StarlarkWriter {
	return &StarlarkWriter{bufio.NewWriter(w), nil, false, "", nil}
}

// BeginMacro starts writing a new macro with the given name.
func (sw *StarlarkWriter) BeginMacro(name string) error {
	if sw.currentMacro != "" {
		return errors.New("nested macros are not allowed")
	}
	sw.buf = append(sw.buf, fmt.Sprintf("def %s(ctx):\n", name))
	sw.currentMacro = name
	sw.hasBody = false
	return nil
}

// EndMacro ends writing the current macro; flushing any pending output.
func (sw *StarlarkWriter) EndMacro() error {
	if sw.currentMacro == "" {
		return errors.New("no current macro")
	}
	err := sw.writeBuffered()
	if err != nil {
		return err
	}
	if !sw.hasBody {
		if err := sw.writeString(sw.indentf("pass\n")); err != nil {
			return err
		}
	}
	sw.currentMacro = ""
	return sw.w.Flush()
}

// PushDirectory writes a Starlark directive indicating a new directory context should be used in the given path.
func (sw *StarlarkWriter) PushDirectory(path string) error {
	if sw.currentMacro == "" {
		return errors.New("no current macro")
	}
	sw.dirStack = append(sw.dirStack, path)
	sw.buf = append(sw.buf, sw.enterString(path))
	return nil
}

func (sw *StarlarkWriter) enterString(path string) string {
	return sw.indentf("ctx = ctx.push_directory(ctx, %#v)\n", path)
}

// PopDirectory writes a Starlark directive indicating that the directory has been exited and to restore the previous context.
func (sw *StarlarkWriter) PopDirectory() (string, error) {
	if sw.currentMacro == "" {
		return "", errors.New("no current macro")
	}
	if len(sw.dirStack) == 0 {
		return "", errors.New("no current directory")
	}
	path := pop(&sw.dirStack)
	// Suppress enter/exit pairs which are otherwise empty.
	if len(sw.buf) > 0 && sw.buf[len(sw.buf)-1] == sw.enterString(path) {
		sw.buf = sw.buf[:len(sw.buf)-1]
		return path, nil
	}
	return path, sw.writeString(sw.indentf("ctx = ctx.pop_directory(ctx)\n"))
}

// WriteCommand writes an invocation of the provided command and arguments.
func (sw *StarlarkWriter) WriteCommand(cmd string, args ...string) error {
	if sw.currentMacro == "" {
		return errors.New("no current macro")
	}
	err := sw.writeBuffered()
	if err != nil {
		return err
	}
	if err := sw.writeString(sw.indentf("ctx.%s(ctx", cmd)); err != nil {
		return err
	}
	for _, arg := range args {
		if err := sw.writeString(fmt.Sprintf(", %#v", arg)); err != nil {
			return err
		}
	}
	return sw.writeString(")\n")
}

func (sw *StarlarkWriter) indentf(format string, vals ...interface{}) string {
	return fmt.Sprintf("    "+format, vals...)
}

func (sw *StarlarkWriter) writeString(s string) error {
	_, err := sw.w.WriteString(s)
	if err == nil && !strings.HasPrefix(s, "def ") {
		sw.hasBody = true
	}
	return err
}

func (sw *StarlarkWriter) writeBuffered() error {
	for _, entry := range sw.buf {
		if err := sw.writeString(entry); err != nil {
			return err
		}
	}
	sw.buf = nil
	return nil
}

func pop(s *[]string) (x string) {
	x, *s = (*s)[len(*s)-1], (*s)[:len(*s)-1]
	return
}
