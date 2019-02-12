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

// Package lexer implements a participle Lexer for the CMakeLists.txt language.
// See https://cmake.org/cmake/help/v3.0/manual/cmake-language.7.html for more details.
package lexer

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/alecthomas/participle/lexer"
)

const (
	_ rune = lexer.EOF - iota
	Space
	Newline
	EscapeSequence
	Quote
	Quoted
	BracketOpen
	BracketContent
	BracketClose
	VarOpen
	VarClose
	Unquoted
	Punct
	Comment
)

type tokenDefinition struct {
	kind rune
	name string
	pat  string
}

// Ordered token definitions. Regular expressions are matched in order!
// Empty patterns are omitted.
var tokenDefs = []tokenDefinition{
	{lexer.EOF, "EOF", ``},
	{Space, "Space", `[ \t]+`},
	{Newline, "Newline", `\n+`},
	{EscapeSequence, "EscapeSequence", `(?s:\\.)`},
	{Quote, "Quote", `"`},
	{Quoted, "Quoted", ``},
	{BracketOpen, "BracketOpen", `\[=*\[`},
	{BracketContent, "BracketContent", ``},
	{BracketClose, "BracketClose", `]=*]`},
	{VarOpen, "VarOpen", `\$\w*\{`},
	{VarClose, "VarClose", `}`},
	{Punct, "Punct", `[()]`},
	{Comment, "Comment", `#`},
	// Unquoted is the last match, so can start with a broader set of characters,
	// but cannot consume additional characters which would match above.
	// We could also only every have it match a single character at a time.
	{Unquoted, "Unquoted", `[^\s()#"][^][\$\s()#"\\}]*`},
}

var (
	eolBytes   = []byte("\n")
	tokenSyms  = make(map[string]rune)
	tokenNames = make(map[rune]string)
	lexPattern *regexp.Regexp
)

func init() {
	var parts []string
	for _, def := range tokenDefs {
		if len(def.pat) > 0 {
			parts = append(parts, fmt.Sprintf(`(?P<%s>%s)`, def.name, regexp.MustCompile(def.pat)))
		}
		tokenSyms[def.name] = def.kind
		tokenNames[def.kind] = def.name
	}
	lexPattern = regexp.MustCompile(strings.Join(parts, "|"))
}

// New returns a new lexer.Definition suitable for lexing CMakeLists.txt
func New() lexer.Definition {
	return &cmakeDefinition{}
}

type cmakeDefinition struct{}

// Lex implements lexer.Definition for CMakeLists.
func (cmakeDefinition) Lex(reader io.Reader) (lexer.Lexer, error) {
	return &filterLexer{l: newScanner(reader)}, nil
}

// Symbols implements lexer.Definition for CMakeLists.
func (cmakeDefinition) Symbols() map[string]rune {
	return tokenSyms
}

type scanner struct {
	s   *bufio.Scanner
	pos lexer.Position // Starting position of the *next* token.
	tok lexer.Token    // Current token.
}

func newScanner(r io.Reader) *scanner {
	scan := &scanner{bufio.NewScanner(r), lexer.Position{
		Filename: lexer.NameOfReader(r),
		Line:     1,
		Column:   1,
	}, lexer.Token{}}
	scan.s.Split(scan.scanPattern)
	return scan
}

// Next implements lexer.Lexer for *scanner and returns the raw token from scanning the input reader.
func (s *scanner) Next() (lexer.Token, error) {
	for s.scan() {
		return s.tok, s.s.Err()
	}
	return lexer.EOFToken(s.pos), nil
}

// scan scans the input to find the next token.
func (s *scanner) scan() bool {
	if s.s.Scan() {
		s.tok.Pos = s.pos
		s.tok.Value = s.s.Text()
		s.updatePosition(s.s.Bytes())
		return true
	}
	return false
}

// updatePosition updates the lexer's position from data.
func (s *scanner) updatePosition(data []byte) lexer.Position {
	s.pos.Offset += len(data)
	lines := bytes.Count(data, eolBytes)
	s.pos.Line += lines
	if lines == 0 {
		s.pos.Column += utf8.RuneCount(data)
	} else {
		s.pos.Column = utf8.RuneCount(data[bytes.LastIndex(data, eolBytes):])
	}
	return s.pos
}

// scanPattern is the bufio.SplitFunc used to partition the input text into tokens.
func (s *scanner) scanPattern(data []byte, atEOF bool) (int, []byte, error) {
	if atEOF && len(data) == 0 {
		s.tok.Type = lexer.EOF
		return 0, nil, nil
	}
	matches := lexPattern.FindSubmatchIndex(data)
	if matches == nil || matches[0] != 0 {
		rn, _ := utf8.DecodeRune(data)
		return 0, nil, lexer.Errorf(s.pos, "invalid token %q", rn)
	}
	for i := 2; i < len(matches); i += 2 {
		if matches[i] != -1 {
			tok := lexPattern.SubexpNames()[i/2]
			s.tok.Type = tokenSyms[tok]
			break
		}
	}
	// We matched all of the input, but aren't at the end. Request more data.
	if !atEOF && len(data) == matches[1]-matches[0] {
		return 0, nil, nil
	}
	return matches[1], data[matches[0]:matches[1]], nil
}
