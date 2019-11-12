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

package lexer

import (
	"strings"
	"testing"

	"github.com/alecthomas/participle/lexer"
	plex "github.com/alecthomas/participle/lexer"
	"github.com/google/go-cmp/cmp"
)

type Token = plex.Token

func newToken(kind rune, value string) Token {
	return Token{Type: kind, Value: value}
}

func newTokenAt(kind rune, value string, offset, line, col int) Token {
	return Token{Type: kind,
		Value: value,
		Pos: lexer.Position{
			Filename: "",
			Offset:   offset,
			Line:     line,
			Column:   col,
		},
	}
}

func lexString(value string) ([]Token, error) {
	lexer, err := New().Lex(strings.NewReader(value))
	if err != nil {
		return nil, err
	}
	return plex.ConsumeAll(lexer)
}

func ignorePosition() cmp.Option {
	return cmp.FilterPath(func(p cmp.Path) bool {
		f, ok := p.Last().(cmp.StructField)
		if !ok {
			return false
		}
		return f.Name() == "Pos"
	}, cmp.Ignore())
}

func removeWhitespace(toks []lexer.Token) []lexer.Token {
	var r []lexer.Token
	for _, tok := range toks {
		switch tok.Type {
		case Space, Newline:
			continue
		default:
			r = append(r, tok)
		}
	}
	return r
}

func TestDirectiveSpacing(t *testing.T) {
	input := []string{
		" directive ( ) ",
		"\ndirective ( ) ",
		"\ndirective\n( ) ",
		"\ndirective\n(\n) ",
		"\ndirective\n(\n)\n",
		"directive\n(\n)\n",
		"directive(\n)\n",
		"directive()\n",
		"directive()",
		"directive #\n()",
		"#\ndirective()",
		"directive(#\n)",
		"directive(#[[comment]])",
		"#[[comment]]directive()",
		"directive#[[comment]]()",
		"directive#[[comment\n]]()",
	}

	expected := []Token{
		{Type: Identifier, Value: "directive"},
		{Type: Punct, Value: "("},
		{Type: Punct, Value: ")"},
		plex.EOFToken(plex.Position{}),
	}

	for _, test := range input {
		tokens, err := lexString(test)
		if err != nil {
			t.Errorf("Error lexing %#v: %s", test, err)
			continue
		}
		tokens = removeWhitespace(tokens)
		if diff := cmp.Diff(tokens, expected, ignorePosition()); diff != "" {
			t.Errorf("Unexpected lex (%#v):\n%s", test, diff)
		}
	}
}

func TestBracketArgument(t *testing.T) {
	tests := map[string][]lexer.Token{
		`[[]]`: { // Empty
			newToken(BracketContent, ``),
		},
		`[=[]=]`: { // Empty, non-empty delimiter.
			newToken(BracketContent, ``),
		},
		`[=[${var}]=]`: { // Unevaluated variable reference.
			newToken(BracketContent, `${var}`),
		},
		`[===[content\n]]]=]]==]]===]`: { // Unmatched delimiters.
			newToken(BracketContent, `content\n]]]=]]==]`),
		},
	}
	for input, expected := range tests {
		tokens, err := lexString(input)
		if err != nil {
			t.Errorf("Error lexing %s: %s", input, err)
			continue
		}
		if diff := cmp.Diff(tokens, append(expected, plex.EOFToken(plex.Position{})), ignorePosition()); diff != "" {
			t.Errorf("Unexpected lex (%#v):\n%s", input, diff)
		}
	}
}

func TestQuotedArgument(t *testing.T) {
	inputs := []string{
		`""`,                // Empty.
		"\"\n\"",            // Newline.
		"\"a\\\nb\"",        // Escaped continuation.
		`"\n"`,              // Newline escape sequence.
		`"\\n"`,             // Escaped newline.
		`"regular text"`,    // Boring regular text.
		`"ident"`,           // Thing that could be an identifier.
		`"\${var}"`,         // Escaped variable reference.
		`"${var}"`,          // Variable reference.
		`"Nested${var}Ref"`, // Nested variable reference.
		// TODO(shahms): Handle mis-quoted-string errors better.
		//`"no end`,        // Missing the closing quote.
	}
	expected := [][]Token{
		{newToken(Quote, `"`), newToken(Quoted, ""), newToken(Quote, `"`)},
		{newToken(Quote, `"`), newToken(Quoted, "\n"), newToken(Quote, `"`)},
		{newToken(Quote, `"`), newToken(Quoted, "ab"), newToken(Quote, `"`)},
		{newToken(Quote, `"`), newToken(EscapeSequence, `\n`), newToken(Quote, `"`)},
		{newToken(Quote, `"`), newToken(EscapeSequence, `\\`), newToken(Quoted, "n"), newToken(Quote, `"`)},
		{newToken(Quote, `"`), newToken(Quoted, "regular text"), newToken(Quote, `"`)},
		{newToken(Quote, `"`), newToken(Quoted, "ident"), newToken(Quote, `"`)},
		{newToken(Quote, `"`), newToken(EscapeSequence, `\$`), newToken(Quoted, "{var"), newToken(VarClose, "}"), newToken(Quote, `"`)},
		{newToken(Quote, `"`), newToken(VarOpen, "${"), newToken(Quoted, "var"), newToken(VarClose, "}"), newToken(Quote, `"`)},
		{
			newToken(Quote, `"`),
			newToken(Quoted, "Nested"),
			newToken(VarOpen, "${"),
			newToken(Quoted, "var"),
			newToken(VarClose, "}"),
			newToken(Quoted, "Ref"),
			newToken(Quote, `"`),
		},
	}
	for n, input := range inputs {
		tokens, err := lexString(input)
		if err != nil {
			t.Errorf("Error lexing %s: %s", input, err)
			continue
		}
		if diff := cmp.Diff(tokens, append(expected[n], plex.EOFToken(plex.Position{})), ignorePosition()); diff != "" {
			t.Errorf("Unexpected lex (%#v):\n%s", input, diff)
		}
	}
}

func TestUnquotedArgument(t *testing.T) {
	tests := map[string][]Token{
		`NoSpace`: {newToken(Identifier, "NoSpace")},
		`Escaped\ Space`: {
			newToken(Unquoted, `Escaped`),
			newToken(EscapeSequence, `\ `),
			newToken(Unquoted, `Space`),
		},
		`This;Divides;Into;Five;Arguments`: {newToken(Unquoted, `This;Divides;Into;Five;Arguments`)},
		`Escaped\;Semicolon`: {
			newToken(Unquoted, `Escaped`),
			newToken(EscapeSequence, `\;`),
			newToken(Unquoted, `Semicolon`),
		},
		`\${var}`: {
			newToken(EscapeSequence, `\$`),
			newToken(Unquoted, "{var"),
			newToken(VarClose, "}"),
		},
		`${var}`: {
			newToken(VarOpen, "${"),
			newToken(Unquoted, "var"),
			newToken(VarClose, "}"),
		},
		`Nested${var}Ref`: {
			newToken(Unquoted, "Nested"),
			newToken(VarOpen, "${"),
			newToken(Unquoted, "var"),
			newToken(VarClose, "}"),
			newToken(Unquoted, "Ref"),
		},
		`A AND NOT(B OR C)`: {
			newToken(Identifier, "A"),
			newToken(Space, " "),
			newToken(Identifier, "AND"),
			newToken(Space, " "),
			newToken(Identifier, "NOT"),
			newToken(Punct, "("),
			newToken(Identifier, "B"),
			newToken(Space, " "),
			newToken(Identifier, "OR"),
			newToken(Space, " "),
			newToken(Identifier, "C"),
			newToken(Punct, ")"),
		},
		`Legacy"em bedded"Quotes`: {newToken(Unquoted, `Legacy"em bedded"Quotes`)},
	}
	// Variable references and escape sequences are handled during evaluation.
	for input, expected := range tests {
		tokens, err := lexString(input)
		if err != nil {
			t.Errorf("Error lexing %s: %s", input, err)
			continue
		}
		if diff := cmp.Diff(tokens, append(expected, plex.EOFToken(plex.Position{})), ignorePosition()); diff != "" {
			t.Errorf("Unexpected lex (%#v):\n%s", input, diff)
		}
	}
}

func TestLexerPosition(t *testing.T) {
	tests := map[string][]Token{
		"directive (\nCOMMAND\n)\n": {
			newTokenAt(Identifier, "directive", 0, 1, 1),
			newTokenAt(Space, " ", 9, 1, 10),
			newTokenAt(Punct, "(", 10, 1, 11),
			newTokenAt(Newline, "\n", 11, 1, 12),
			newTokenAt(Identifier, "COMMAND", 12, 2, 1),
			newTokenAt(Newline, "\n", 19, 2, 8),
			newTokenAt(Punct, ")", 20, 3, 1),
			newTokenAt(Newline, "\n", 21, 3, 2),
			newTokenAt(plex.EOF, "", 22, 4, 1),
		},
	}
	for input, expected := range tests {
		tokens, err := lexString(input)
		if err != nil {
			t.Errorf("Error parsing %s: %s", input, err)
			continue
		}
		if diff := cmp.Diff(tokens, expected); diff != "" {
			t.Errorf("Unexpected lex (%#v):\n%s", input, diff)
		}

	}
}

func TestLexerMixedArguments(t *testing.T) {
	tests := map[string][]Token{
		`directive(1234 Unquoted;List Nested${VAR}Ref "Quoted${VAR}Ref")`: {
			newToken(Identifier, "directive"),
			newToken(Punct, "("),
			newToken(Unquoted, "1234"),
			newToken(Space, " "),
			newToken(Unquoted, "Unquoted;List"),
			newToken(Space, " "),
			newToken(Unquoted, "Nested"),
			newToken(VarOpen, "${"),
			newToken(Unquoted, "VAR"),
			newToken(VarClose, "}"),
			newToken(Unquoted, "Ref"),
			newToken(Space, " "),
			newToken(Quote, `"`),
			newToken(Quoted, "Quoted"),
			newToken(VarOpen, "${"),
			newToken(Quoted, "VAR"),
			newToken(VarClose, "}"),
			newToken(Quoted, "Ref"),
			newToken(Quote, `"`),
			newToken(Punct, ")"),
			newToken(plex.EOF, ""),
		},
		`directive(terrible"cho#ces"tail)`: {
			newToken(Identifier, "directive"),
			newToken(Punct, "("),
			newToken(Identifier, "terrible"),
			newToken(Quote, `"`),
			newToken(Quoted, "cho#ces"),
			newToken(Quote, `"`),
			newToken(Identifier, "tail"),
			newToken(Punct, ")"),
			newToken(plex.EOF, ""),
		},
	}
	for input, expected := range tests {
		tokens, err := lexString(input)
		if err != nil {
			t.Errorf("Error parsing %s: %s", input, err)
			continue
		}
		if diff := cmp.Diff(tokens, expected, ignorePosition()); diff != "" {
			t.Errorf("Unexpected lex (%#v):\n%s", input, diff)
		}
	}
}
