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

// StartCondition indicates a particular lexer state in which a rule should apply.
// By default, start conditions are inclusive and will match rules belonging to an empty
// set of start conditions as well as those which are explicitly specified.
// Exclusive start conditions only match if the scanner is in the indicated state.
type StartCondition int

const (
	InitialCondition StartCondition = 0  // Initial start condition for a scanner.
	EOFPattern                      = `` // String indicating a rule should match at EOF.

)

var EOFRegexp *regexp.Regexp // Regexp indicating a rule should match at EOF.

// ScanState interface defines a minimal set of behaviors expected by an action callback.
type ScanState interface {
	Begin(StartCondition) // Transition the ScanState to the indicating start condition.
	Bytes() []byte        // The currently matched bytes.
	Token() *lexer.Token  // The lexer.Token being constructed.
}

// Action is a callback intended to be invoked when the rule conditions match.
type Action func(ScanState) (bool, error)

// Rules is a collection of rules to match against an incoming text string and current StartCondtion.
type Rules struct {
	condMap map[StartCondition]bool
	table   []rule
}

// rule is a single entry, indicating a list of start conditions and pattern to select an action.
type rule struct {
	conds  []StartCondition
	re     *regexp.Regexp
	action Action
}

// ruleBuilder abstracts start condtion collection to make rule table definitions more readable.
type ruleBuilder struct {
	conds []StartCondition
}

// Option is a callback to apply to the Rules object during construction.
type Option func(*Rules)

// ExclusiveConditions configures the Rules table so the provided StartConditions are considered exclusive.
func ExclusiveConditions(cond StartCondition, tail ...StartCondition) Option {
	return func(r *Rules) {
		r.condMap[cond] = true
		for _, cond := range tail {
			r.condMap[cond] = true
		}
	}
}

// InclusiveConditions configures the Rules table so the provided StartConditions are considered inclusive (the default).
func InclusiveConditions(cond StartCondition, tail ...StartCondition) Option {
	return func(r *Rules) {
		r.condMap[cond] = false
		for _, cond := range tail {
			r.condMap[cond] = false
		}
	}
}

// In accepts a (possibly empty) list of start conditions during which to consider a rule.
func In(conds ...StartCondition) *ruleBuilder {
	return &ruleBuilder{conds}
}

// Match returns an option which adds the configured rule to the rules table.
func (c *ruleBuilder) Match(pat string, action Action) Option {
	return func(r *Rules) {
		r.MustAdd(c.conds, pat, action)
	}
}

// New returns a new Rules table, after applying the provided options.
func New(opts ...Option) *Rules {
	r := &Rules{make(map[StartCondition]bool), nil}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// AddRegexp adds a rule matching the regular expression and start conditions.
func (r *Rules) AddRegexp(conds []StartCondition, re *regexp.Regexp, action Action) error {
	r.table = append(r.table, rule{conds, re, action})
	return nil
}

// Add adds a rule matching the pattern and start conditions.
func (r *Rules) Add(conds []StartCondition, pat string, action Action) error {
	re, err := compileRegexp(pat)
	if err != nil {
		return err
	}
	return r.AddRegexp(conds, re, action)
}

// Add adds a rule matching the pattern and start conditions.
func (r *Rules) MustAdd(conds []StartCondition, pat string, action Action) {
	r.AddRegexp(conds, mustCompileRegexp(pat), action)
}

// Match considers applicable rules and returns the action associated with the longest
// matching pattern, as well as the portion of the data matched by that pattern.
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
