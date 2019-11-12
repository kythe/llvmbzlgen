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
	"io"

	"github.com/alecthomas/participle/lexer"
)

// Constants defining the token types used by CMake
const (
	_ rune = lexer.EOF - iota
	Space
	Newline
	EscapeSequence
	Quoted
	Quote
	BracketContent
	BracketComment
	VarOpen
	VarClose
	Identifier
	Unquoted
	Punct
	Comment
)

var (
	tokenSyms = map[string]rune{
		"EOF":            lexer.EOF,
		"Space":          Space,
		"Newline":        Newline,
		"EscapeSequence": EscapeSequence,
		"Quote":          Quote,
		"Quoted":         Quoted,
		"BracketContent": BracketContent,
		"BracketComment": BracketComment,
		"VarOpen":        VarOpen,
		"VarClose":       VarClose,
		"Identifier":     Identifier,
		"Unquoted":       Unquoted,
		"Punct":          Punct,
		"Comment":        Comment,
	}
	tokenNames = make(map[rune]string)
)

func init() {
	for name, kind := range tokenSyms {
		tokenNames[kind] = name
	}
}

// New returns a new lexer.Definition suitable for lexing CMakeLists.txt
func New() lexer.Definition {
	return &cmakeDefinition{}
}

type cmakeDefinition struct{}

// Lex implements lexer.Definition for CMakeLists.
func (cmakeDefinition) Lex(reader io.Reader) (lexer.Lexer, error) {
	return newSplitLexer(reader), nil
}

// Symbols implements lexer.Definition for CMakeLists.
func (cmakeDefinition) Symbols() map[string]rune {
	return tokenSyms
}
