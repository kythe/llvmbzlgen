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

package rules

import (
	"bufio"
	"bytes"
	"io"
	"unicode/utf8"

	"github.com/alecthomas/participle/lexer"
)

var (
	eolBytes = []byte("\n")
)

type Scanner struct {
	rules *Rules
	s     *bufio.Scanner

	pos  lexer.Position
	cond StartCondition

	action Action
}

func NewScanner(rules *Rules, r io.Reader) *Scanner {
	s := &Scanner{
		rules,
		bufio.NewScanner(r),
		lexer.Position{
			Filename: lexer.NameOfReader(r),
			Offset:   0,
			Line:     1,
			Column:   1,
		},
		InitialCondition,
		nil,
	}
	s.s.Split(s.splitRules)
	return s
}

func (s *Scanner) Begin(cond StartCondition) {
	s.cond = cond
}

func (s *Scanner) SetPosition(pos lexer.Position) {
	s.pos = pos
}

func (s *Scanner) Scan() bool {
	if s.s.Scan() {
		updatePosition(&s.pos, s.s.Bytes())
		return true
	}
	return false
}

func (s *Scanner) Pos() lexer.Position {
	return s.pos
}

func (s *Scanner) Action() Action {
	return s.action
}

func (s *Scanner) Bytes() []byte {
	return s.s.Bytes()
}

func (s *Scanner) Err() error {
	return s.s.Err()
}

func (s *Scanner) splitRules(data []byte, atEOF bool) (int, []byte, error) {
	if action, token := s.rules.Match(s.cond, data); action == nil {
		s.action = nil
		rn, _ := utf8.DecodeRune(data)
		return 0, nil, lexer.Errorf(s.pos, "invalid token %q", rn)
	} else if !atEOF && len(data) == len(token) {
		// We matched the entirety of the input, request more data.
		return 0, nil, nil
	} else {
		s.action = action
		return len(token), token, nil
	}
}

// updatePosition updates the position from data.
func updatePosition(pos *lexer.Position, data []byte) {
	pos.Offset += len(data)
	lines := bytes.Count(data, eolBytes)
	pos.Line += lines
	if lines == 0 {
		pos.Column += utf8.RuneCount(data)
	} else {
		pos.Column = utf8.RuneCount(data[bytes.LastIndex(data, eolBytes):])
	}
}
