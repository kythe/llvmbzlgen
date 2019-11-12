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
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strings"
	"unicode/utf8"

	"github.com/alecthomas/participle/lexer"
	"github.com/kythe/llvmbzlgen/cmakelib/lexer/rules"
)

const (
	initialCondition rules.StartCondition = rules.InitialCondition + iota
	commentCondition
	bracketCondition
	bracketEndCondition
	stringCondition
)

var (
	makeVarPattern  = `\$\([A-Za-z0-9_]*\)`
	unquotedPattern = `([^ \0\t\r\n()#\\"[=]|\\[^\0\n])`
	legacyPattern   = fmt.Sprintf(`(%s|%s|"(%s|%s|[ \t[=])*")`, makeVarPattern, unquotedPattern, makeVarPattern, unquotedPattern)
)

var fileTable = rules.New(
	rules.ExclusiveConditions(
		commentCondition,
		bracketCondition,
		bracketEndCondition,
		stringCondition,
	),
	rules.In(rules.InitialCondition, commentCondition).Match(`\n`, lexNewline),
	rules.In().Match(`#?\[=*\[\n?`, lexBracketOpen),
	rules.In().Match(`#`, lexCommentStart),
	rules.In(commentCondition).Match(`[^\0\n]*`, lexComment),
	rules.In().Match(`[()]`, lexParen),
	rules.In().Match(`[A-Zaa-z_][A-Za-z0-9_]*`, lexIdentifier),
	rules.In(bracketCondition).Match(`\]=*`, lexBracketTail),
	rules.In(bracketEndCondition).Match(`\]`, lexBracketClose),
	rules.In(bracketCondition).Match(`([^]\0\n])+`, lexBracketContent),
	rules.In(bracketCondition, bracketEndCondition).Match(`\n`, lexBracketReset),
	rules.In(bracketCondition, bracketEndCondition).Match(`[^\0\n]`, lexBracketReset),
	rules.In(bracketCondition, bracketEndCondition).Match(rules.EOFPattern, lexBracketEOF),
	rules.In().Match(fmt.Sprintf(`(%s|=|\[=*%s)(%s|[[=])*`, unquotedPattern, unquotedPattern, unquotedPattern), lexUnquoted),
	rules.In().Match(fmt.Sprintf(`(%s|%s|=|\[=*%s)(%s|[[=])*`, makeVarPattern, unquotedPattern, legacyPattern, legacyPattern), lexUnquoted),
	rules.In().Match(`\[`, lexUnquoted),
	rules.In().Match(`"`, lexOpenQuote),
	rules.In(stringCondition).Match(`([^\\\0\n"]|\\[^\0\n])+`, lexQuoted),
	rules.In(stringCondition).Match(`\\\n`, lexContinuation),
	rules.In(stringCondition).Match(`\n`, lexQuoted),
	rules.In(stringCondition).Match(`"`, lexEndQuote),
	rules.In(stringCondition).Match(`[^\0\n]`, lexQuoted),
	rules.In(stringCondition).Match(rules.EOFPattern, lexQuotedEOF),
	rules.In().Match(`[ \t\r]+`, lexSpace),
	rules.In().Match(`.`, lexUnexpected),
	rules.In().Match(rules.EOFPattern, lexEOF),
)

var argTable = rules.New(
	rules.In().Match(`\$ENV\{`, lexEnvOpen),
	rules.In().Match(`\$[A-Za-z0-9_.+-]*\{`, lexVarOpen),
	rules.In().Match(`}`, lexVarClose),
	rules.In().Match(`\\.`, lexEscapeSequence),
	rules.In().Match(`[^$\\}]+`, lexArgument),
	rules.In().Match(`.`, lexArgument),
	rules.In().Match(rules.EOFPattern, lexEOF),
)

type tableLexer struct {
	s *rules.Scanner

	buf []lexer.Token

	bracket int
	base    lexer.Token
}

type driver tableLexer

type splitLexer struct {
	file lexer.Lexer
	arg  lexer.Lexer
}

func (s *splitLexer) Next() (lexer.Token, error) {
	if s.arg != nil {
		if next, err := s.arg.Next(); !(err == nil && next.Type == lexer.EOF) {
			return next, err
		}
		s.arg = nil
	}
	next, err := s.file.Next()
	if err != nil {
		return next, err
	}
	switch next.Type {
	case Quoted, Unquoted:
		arg := newArgumentLexer(next)
		// Short-circuit the argument lexer if the argument is "empty" or a single token.
		if succ, err := arg.Next(); err != nil || (succ.Type != lexer.EOF && !reflect.DeepEqual(succ, next)) {
			s.arg = arg
			return succ, err
		}
	}
	return next, err
}

func newFileLexer(r io.Reader) *tableLexer {
	return &tableLexer{
		rules.NewScanner(fileTable, r),
		nil,
		-1,
		lexer.Token{},
	}
}

func newArgumentLexer(base lexer.Token) *tableLexer {
	l := &tableLexer{
		rules.NewScanner(argTable, strings.NewReader(base.Value)),
		nil,
		-1,
		base,
	}
	l.s.SetPosition(base.Pos)
	return l
}

func newSplitLexer(r io.Reader) *splitLexer {
	return &splitLexer{newFileLexer(r), nil}
}

func (l *tableLexer) Next() (lexer.Token, error) {
	for {
		if len(l.buf) > 0 {
			tok := l.buf[0]
			l.buf = l.buf[1:]
			return tok, nil
		}
		if err := l.advance(); err != nil {
			return lexer.EOFToken(l.s.Pos()), err
		}
	}
}

func (l *tableLexer) advance() error {
	// Reset the token.
	l.buf = []lexer.Token{lexer.EOFToken(l.s.Pos())}
	for l.s.Scan() {
		if done, err := l.s.Action()((*driver)(l)); done || err != nil {
			return err
		}
	}
	if l.s.Err() != nil {
		return l.s.Err()
	}
	if l.s.Action() != nil {
		_, err := l.s.Action()((*driver)(l))
		if err != nil {
			return err
		}
	}
	return l.s.Err()
}

func (d *driver) Begin(cond rules.StartCondition) {
	d.s.Begin(cond)
}

func (d *driver) Bytes() []byte {
	return d.s.Bytes()
}

func (d *driver) Token() *lexer.Token {
	return &d.buf[len(d.buf)-1]
}

func lexNewline(d rules.ScanState) (bool, error) {
	setValue(d.Token(), Newline, string(d.Bytes()))
	d.Begin(initialCondition)
	return true, nil
}

func lexCommentStart(d rules.ScanState) (bool, error) {
	d.Begin(commentCondition)
	return false, nil
}

func lexComment(d rules.ScanState) (bool, error) {
	return false, nil
}

func lexParen(d rules.ScanState) (bool, error) {
	setValue(d.Token(), Punct, string(d.Bytes()))
	return true, nil
}

func lexIdentifier(d rules.ScanState) (bool, error) {
	setValue(d.Token(), Identifier, string(d.Bytes()))
	return true, nil
}

func lexBracketOpen(d rules.ScanState) (bool, error) {
	text := d.Bytes()
	if text[0] == '#' {
		setValue(d.Token(), BracketComment, "")
	} else {
		setValue(d.Token(), BracketContent, "")
	}
	d.Begin(bracketCondition)
	l := d.(*driver)
	l.bracket = bytes.LastIndexByte(text, '[') - bytes.IndexByte(text, '[')
	return false, nil
}

func lexBracketTail(d rules.ScanState) (bool, error) {
	appendText(d.Token(), string(d.Bytes()))
	l := d.(*driver)
	if len(d.Bytes()) == l.bracket {
		d.Begin(bracketEndCondition)
	}
	return false, nil
}

func lexBracketClose(d rules.ScanState) (bool, error) {
	tok := d.Token()
	l := d.(*driver)
	l.Begin(initialCondition)
	tok.Value = tok.Value[0 : len(tok.Value)-l.bracket]
	if tok.Type == BracketComment {
		return false, nil
	}
	return true, nil
}

func lexBracketContent(d rules.ScanState) (bool, error) {
	appendText(d.Token(), string(d.Bytes()))
	return false, nil
}

func lexBracketReset(d rules.ScanState) (bool, error) {
	d.Begin(bracketCondition)
	return lexBracketContent(d)
}

func lexBracketEOF(d rules.ScanState) (bool, error) {
	d.Begin(initialCondition)
	return true, lexer.Errorf(d.Token().Pos, "unterminated bracket with text: %s", d.Token().Value)
}

func lexUnquoted(d rules.ScanState) (bool, error) {
	setValue(d.Token(), Unquoted, string(d.Bytes()))
	return true, nil
}

func lexOpenQuote(d rules.ScanState) (bool, error) {
	setValue(d.Token(), Quote, `"`)
	l := d.(*driver)
	l.buf = append(l.buf, lexer.Token{
		Pos:   l.s.Pos(),
		Type:  Quoted,
		Value: "",
	})
	d.Begin(stringCondition)
	return false, nil
}

func lexQuoted(d rules.ScanState) (bool, error) {
	appendText(d.Token(), string(d.Bytes()))
	return false, nil
}

func lexContinuation(d rules.ScanState) (bool, error) {
	return false, nil
}

func lexEndQuote(d rules.ScanState) (bool, error) {
	d.Begin(initialCondition)
	l := d.(*driver)
	l.buf = append(l.buf, lexer.Token{
		Pos:   l.s.Pos(),
		Type:  Quote,
		Value: `"`,
	})
	return true, nil
}

func lexQuotedEOF(d rules.ScanState) (bool, error) {
	d.Begin(initialCondition)
	return true, lexer.Errorf(d.Token().Pos, "unterminated string with value: %q", d.Token().Value)
}

func lexSpace(d rules.ScanState) (bool, error) {
	setValue(d.Token(), Space, string(d.Bytes()))
	return true, nil
}

func lexUnexpected(d rules.ScanState) (bool, error) {
	rn, _ := utf8.DecodeRune(d.Bytes())
	return true, lexer.Errorf(d.Token().Pos, "invalid token %q", rn)
}

func lexEOF(d rules.ScanState) (bool, error) {
	*d.Token() = lexer.EOFToken(d.Token().Pos)
	return true, nil
}

func lexEnvOpen(d rules.ScanState) (bool, error) {
	return lexVarOpen(d)
}

func lexVarOpen(d rules.ScanState) (bool, error) {
	setValue(d.Token(), VarOpen, string(d.Bytes()))
	return true, nil
}

func lexVarClose(d rules.ScanState) (bool, error) {
	setValue(d.Token(), VarClose, string(d.Bytes()))
	return true, nil
}

func lexEscapeSequence(d rules.ScanState) (bool, error) {
	setValue(d.Token(), EscapeSequence, string(d.Bytes()))
	return true, nil
}

func lexArgument(d rules.ScanState) (bool, error) {
	setValue(d.Token(), d.(*driver).base.Type, string(d.Bytes()))
	return true, nil
}

func setValue(t *lexer.Token, kind rune, value string) {
	t.Type = kind
	t.Value = value
}

func appendText(t *lexer.Token, value string) {
	t.Value += value
}
