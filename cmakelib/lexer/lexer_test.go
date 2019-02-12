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
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/alecthomas/participle/lexer"
	plex "github.com/alecthomas/participle/lexer"
	"github.com/alecthomas/repr"
)

type Token = plex.Token

func NewToken(kind rune, value string) Token {
	return Token{Type: kind, Value: value}
}

func NewTokenAt(kind rune, value string, offset, line, col int) Token {
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

func compareTokens(a, b []Token) bool {
	if len(a) != len(b) {
		return false
	}
	for n, _ := range a {
		if a[n].Type != b[n].Type || a[n].Value != b[n].Value {
			return false
		}
	}
	return true
}

func lexString(value string) ([]Token, error) {
	lexer, err := New().Lex(strings.NewReader(value))
	if err != nil {
		return nil, err
	}
	return plex.ConsumeAll(lexer)
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
		{Type: Unquoted, Value: "directive"},
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
		if len(tokens) != len(expected) {
			t.Errorf("Invalid lex (%#v): %v != %v", test, expected, tokens)
		}
		for n, tok := range tokens {
			if tok.Type != expected[n].Type || tok.Value != expected[n].Value {
				t.Errorf("Invalid lex (%#v): %v != %v", test, expected[n], tok)
			}
		}
	}
}

func TestBracketArgument(t *testing.T) {
	tests := map[string][]lexer.Token{
		`[[]]`: { // Empty
			NewToken(BracketOpen, `[[`),
			NewToken(BracketClose, `]]`),
		},
		`[=[]=]`: { // Empty, non-empty delimiter.
			NewToken(BracketOpen, `[=[`),
			NewToken(BracketClose, `]=]`),
		},
		`[=[${var}]=]`: { // Unevaluated variable reference.
			NewToken(BracketOpen, `[=[`),
			NewToken(BracketContent, `${var}`),
			NewToken(BracketClose, `]=]`),
		},
		`[===[content\n]]]=]]==]]===]`: { // Unmatched delimiters.
			NewToken(BracketOpen, `[===[`),
			NewToken(BracketContent, `content\n]]]=]]==]`),
			NewToken(BracketClose, `]===]`),
		},
	}
	for input, expected := range tests {
		tokens, err := lexString(input)
		if err != nil {
			t.Errorf("Error parsing %s: %s", input, err)
			continue
		}
		if !compareTokens(tokens, append(expected, plex.EOFToken(plex.Position{}))) {
			t.Errorf("\nExpecting %#v\nfound %#v", expected, tokens)
		}
	}
}

func TestQuotedArgument(t *testing.T) {
	inputs := []string{
		`""`,                // Empty.
		`"\n"`,              // Newline.
		`"\\n"`,             // Escaped newline.
		`"regular text"`,    // Boring regular text.
		`"ident"`,           // Thing that could be an identifier.
		`"\${var}"`,         // Escaped variable reference.
		`"${var}"`,          // Variable reference.
		`"Nested${var}Ref"`, // Nested variable reference.
		// TODO(shahms): Handle mid-quoted-string errors better.
		//`"no end`,        // Missing the closing quote.
	}
	expected := [][]Token{
		{},
		{NewToken(EscapeSequence, `\n`)},
		{NewToken(EscapeSequence, `\\`), NewToken(Quoted, "n")},
		{NewToken(Quoted, "regular text")},
		{NewToken(Quoted, "ident")},
		{NewToken(EscapeSequence, `\$`), NewToken(Quoted, "{var"), NewToken(VarClose, "}")},
		{NewToken(VarOpen, "${"), NewToken(Quoted, "var"), NewToken(VarClose, "}")},
		{
			NewToken(Quoted, "Nested"),
			NewToken(VarOpen, "${"),
			NewToken(Quoted, "var"),
			NewToken(VarClose, "}"),
			NewToken(Quoted, "Ref"),
		},
	}
	for n, input := range inputs {
		tokens, err := lexString(input)
		if err != nil {
			t.Errorf("Error parsing %s: %s", input, err)
			continue
		}
		// Strip of the tokens for the quotation marks and EOF.
		tokens = tokens[1 : len(tokens)-2]
		if !compareTokens(tokens, append(expected[n])) {
			t.Errorf("\nExpecting %#v\nfound %#v", expected[n], tokens)
		}
	}
}

func TestUnquotedArgument(t *testing.T) {
	tests := map[string][]Token{
		`NoSpace`: {NewToken(Unquoted, "NoSpace")},
		`Escaped\ Space`: {
			NewToken(Unquoted, `Escaped`),
			NewToken(EscapeSequence, `\ `),
			NewToken(Unquoted, `Space`),
		},
		`This;Divides;Into;Five;Arguments`: {NewToken(Unquoted, `This;Divides;Into;Five;Arguments`)},
		`Escaped\;Semicolon`: {
			NewToken(Unquoted, `Escaped`),
			NewToken(EscapeSequence, `\;`),
			NewToken(Unquoted, `Semicolon`),
		},
		`\${var}`: {
			NewToken(EscapeSequence, `\$`),
			NewToken(Unquoted, "{var"),
			NewToken(VarClose, "}"),
		},
		`${var}`: {
			NewToken(VarOpen, "${"),
			NewToken(Unquoted, "var"),
			NewToken(VarClose, "}"),
		},
		`Nested${var}Ref`: {
			NewToken(Unquoted, "Nested"),
			NewToken(VarOpen, "${"),
			NewToken(Unquoted, "var"),
			NewToken(VarClose, "}"),
			NewToken(Unquoted, "Ref"),
		},
		`A AND NOT(B OR C)`: {
			NewToken(Unquoted, "A"),
			NewToken(Space, " "),
			NewToken(Unquoted, "AND"),
			NewToken(Space, " "),
			NewToken(Unquoted, "NOT"),
			NewToken(Punct, "("),
			NewToken(Unquoted, "B"),
			NewToken(Space, " "),
			NewToken(Unquoted, "OR"),
			NewToken(Space, " "),
			NewToken(Unquoted, "C"),
			NewToken(Punct, ")"),
		},
		// TODO(shahms): Support this.
		//`Legacy"em bedded"Quotes`: {NewToken(Unquoted, `Legacy"em bedded"Quotes`)},
	}
	// Variable references and escape sequences are handled during evaluation.
	for input, expected := range tests {
		tokens, err := lexString(input)
		if err != nil {
			t.Errorf("Error parsing %s: %s", input, err)
			continue
		}
		if !compareTokens(tokens, append(expected, plex.EOFToken(plex.Position{}))) {
			t.Errorf("\nExpecting %#v\nfound %#v", expected, tokens)
		}
	}
}

func TestLexerPosition(t *testing.T) {
	tests := map[string][]Token{
		"directive (\nCOMMAND\n)\n": {
			NewTokenAt(Unquoted, "directive", 0, 1, 1),
			NewTokenAt(Space, " ", 9, 1, 10),
			NewTokenAt(Punct, "(", 10, 1, 11),
			NewTokenAt(Newline, "\n", 11, 1, 12),
			NewTokenAt(Unquoted, "COMMAND", 12, 2, 1),
			NewTokenAt(Newline, "\n", 19, 2, 8),
			NewTokenAt(Punct, ")", 20, 3, 1),
			NewTokenAt(Newline, "\n", 21, 3, 2),
			NewTokenAt(plex.EOF, "", 22, 4, 1),
		},
	}
	for input, expected := range tests {
		fmt.Println(expected[0].Pos)
		tokens, err := lexString(input)
		if err != nil {
			t.Errorf("Error parsing %s: %s", input, err)
			continue
		}
		if len(tokens) != len(expected) {
			t.Errorf("\nExpected %s\nfound %s",
				repr.String(expected, repr.Indent("  "), repr.OmitEmpty(true)),
				repr.String(tokens, repr.Indent("  "), repr.OmitEmpty(true)))
		}

		for i, tok := range tokens {
			if !reflect.DeepEqual(tok, expected[i]) {
				t.Errorf("\nExpected %s\nfound %s",
					repr.String(expected[i], repr.Indent("  "), repr.OmitEmpty(true), repr.IgnoreGoStringer()),
					repr.String(tok, repr.Indent("  "), repr.OmitEmpty(true), repr.IgnoreGoStringer()))
			}
		}

	}
}
