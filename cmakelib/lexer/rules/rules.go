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

// Package rules implements a flex-like rules for a table driven lexer.
package rules

import (
	"regexp"

	"github.com/alecthomas/participle/lexer"
)

type StartCondition int

const (
	InitialCondition StartCondition = 0
	EOFPattern                      = ``
)

var EOFRegexp *regexp.Regexp

type ScanState interface {
	Begin(StartCondition)
	Bytes() []byte
	Token() *lexer.Token
}

type Action func(ScanState) (bool, error)

type Rules struct {
	condMap map[StartCondition]bool
	table   []rule
}

type rule struct {
	conds  []StartCondition
	re     *regexp.Regexp
	action Action
}

type ruleBuilder struct {
	conds []StartCondition
}

type Option func(*Rules)

func ExclusiveConditions(cond StartCondition, tail ...StartCondition) Option {
	return func(r *Rules) {
		r.condMap[cond] = true
		for _, cond := range tail {
			r.condMap[cond] = true
		}
	}
}

func InclusiveConditions(cond StartCondition, tail ...StartCondition) Option {
	return func(r *Rules) {
		r.condMap[cond] = false
		for _, cond := range tail {
			r.condMap[cond] = false
		}
	}
}

func In(conds ...StartCondition) *ruleBuilder {
	return &ruleBuilder{conds}
}

func (c *ruleBuilder) Match(pat string, action Action) Option {
	return func(r *Rules) {
		r.MustAdd(c.conds, pat, action)
	}
}

func New(opts ...Option) *Rules {
	r := &Rules{make(map[StartCondition]bool), nil}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func (r *Rules) AddRegexp(conds []StartCondition, re *regexp.Regexp, action Action) error {
	r.table = append(r.table, rule{conds, re, action})
	return nil
}

func (r *Rules) Add(conds []StartCondition, pat string, action Action) error {
	re, err := compileRegexp(pat)
	if err != nil {
		return err
	}
	return r.AddRegexp(conds, re, action)
}

func (r *Rules) MustAdd(conds []StartCondition, pat string, action Action) {
	r.AddRegexp(conds, mustCompileRegexp(pat), action)
}

func (r *Rules) Match(curr StartCondition, data []byte) (Action, []byte) {
	var found struct {
		action  Action
		matched []byte
	}
	for _, entry := range r.table {
		if r.matchCondition(curr, entry.conds) {
			// EOF pattern matches at EOF and only at EOF, so take the first.
			if entry.re == EOFRegexp {
				if len(data) == 0 {
					return entry.action, nil
				} else {
					continue
				}
			}
			if locs := entry.re.FindIndex(data); locs != nil && locs[0] == 0 && locs[1] > len(found.matched) {
				found.action = entry.action
				found.matched = data[0:locs[1]]
			}
		}
	}
	return found.action, found.matched
}

func (r *Rules) matchCondition(curr StartCondition, conds []StartCondition) bool {
	if len(conds) == 0 && !r.condMap[curr] {
		return true
	}
	for _, cond := range conds {
		if cond == curr {
			return true
		}
	}
	return false
}

func compileRegexp(pat string) (*regexp.Regexp, error) {
	if pat == EOFPattern {
		return EOFRegexp, nil
	}
	re, err := regexp.Compile(pat)
	if err != nil {
		return re, err
	}
	re.Longest()
	return re, nil
}

func mustCompileRegexp(pat string) *regexp.Regexp {
	if pat == EOFPattern {
		return EOFRegexp
	}
	re := regexp.MustCompile(pat)
	re.Longest()
	return re
}
