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
	"github.com/alecthomas/participle/lexer"
)

// filterLexer processes the raw stream of tokens coming from the underlying scanner-based lexer
// to make the resulting AST simpler and more readable.
type filterLexer struct {
	l    lexer.Lexer
	buf  []lexer.Token
	prev lexer.Token
}

// Next implements the lexer.Lexer interface for filterLexer.
func (l *filterLexer) Next() (lexer.Token, error) {
	var err error
	if len(l.buf) > 0 {
		tok := l.buf[0]
		l.buf = l.buf[1:]
		return tok, nil
	}
	switch l.prev.Type {
	case BracketOpen:
		return l.bufferTokens(combineBracketContent(l.l, len(l.prev.Value)))
	case Quote:
		return l.bufferTokens(combineQuotedContent(l.l))
	}
scan:
	for {
		l.prev, err = l.l.Next()
		if err != nil {
			break
		}
		switch l.prev.Type {
		case Comment:
			l.prev, err = l.bufferTokens(consumeComment(l.l))
			if err != nil || l.prev.Type == lexer.EOF {
				break scan
			}
		default:
			break scan
		}
	}
	if err != nil {
		l.prev = lexer.Token{}
	}
	return l.prev, err
}

// bufferTokens updates the buffer and returns the first buffered token or error.
// If done is true or there is an error, clears the combining token.
func (l *filterLexer) bufferTokens(toks []lexer.Token, done bool, err error) (lexer.Token, error) {
	if done || err != nil {
		l.prev = lexer.Token{}
	}
	if len(toks) > 0 {
		l.buf = toks[1:]
		return toks[0], err
	}
	l.buf = nil
	return lexer.Token{}, err
}

// combineBracketContent merges tokens from the lexer until it encounters
// a BracketClose token with a value equal to hdrlen, EOF or an error.
// The terminating bracket is included in the returned value.
// Returns a slice of the accumulated tokens, whether or not the terminal token was found and any errors encountered.
func combineBracketContent(l lexer.Lexer, hdrlen int) ([]lexer.Token, bool, error) {
	var toks []lexer.Token
	for {
		next, err := l.Next()
		if err != nil {
			return toks, true, err
		}
		switch {
		case next.Type == lexer.EOF, next.Type == BracketClose && len(next.Value) == hdrlen:
			return append(toks, next), true, nil
		case len(toks) == 0:
			toks = append(toks, lexer.Token{
				Type:  BracketContent,
				Pos:   next.Pos,
				Value: next.Value,
			})
		default:
			toks[0].Value += next.Value
		}

	}
	return nil, true, nil
}

// combineQuotedContent reads tokens until it encounters a double-quote or error, merging
// as appropriate.
// Returns a slice of the accumulated tokens, whether or not the terminal token was found and any errors encountered.
func combineQuotedContent(l lexer.Lexer) ([]lexer.Token, bool, error) {
	var toks []lexer.Token
	for {
		next, err := l.Next()
		if err != nil {
			return toks, true, err
		}
		switch next.Type {
		case EscapeSequence, VarOpen, VarClose:
			return append(toks, next), false, nil
		case Quote, lexer.EOF:
			return append(toks, next), true, nil
		default:
			if len(toks) == 0 || toks[len(toks)-1].Type != Quoted {
				toks = append(toks, lexer.Token{
					Type:  Quoted,
					Value: next.Value,
					Pos:   next.Pos,
				})
			} else {
				toks[len(toks)-1].Value += next.Value
			}
		}
	}
	return nil, true, nil
}

// consumeComment reads tokens until it encounters a newline, BracketOpen/BracketClose or EOF.
// Returns a slice of the accumulated tokens, whether or not the terminal token was found and any errors encountered.
func consumeComment(l lexer.Lexer) ([]lexer.Token, bool, error) {
	for {
		next, err := l.Next()
		if err != nil {
			return nil, true, err
		}
		switch next.Type {
		case lexer.EOF, Newline: // Comments do not contain the terminating newline.
			return []lexer.Token{next}, true, nil
		case BracketOpen:
			for {
				toks, done, err := combineBracketContent(l, len(next.Value))
				if err != nil {
					return nil, true, err
				}
				if done && len(toks) > 0 && toks[len(toks)-1].Type == lexer.EOF {
					return toks[len(toks)-1:], true, nil
				}
				return nil, true, nil
			}
		}
	}
}
