/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package filter

import (
	"fmt"

	"github.com/ti-mo/conntrack"
)

// Filter represents a compiled set of filter rules
type Filter struct {
	Rules []CompiledRule
}

// CompiledRule represents a parsed and compiled filter rule
type CompiledRule struct {
	Rule      *Rule
	Predicate PredicateFunc
	RuleText  string
}

// NewFilter creates a new DSL-based filter from rule strings
func NewFilter(ruleStrings []string) (*Filter, error) {
	filter := &Filter{
		Rules: make([]CompiledRule, 0, len(ruleStrings)),
	}

	for i, ruleStr := range ruleStrings {
		parser, err := NewParser(ruleStr)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize parser for rule %d: %w", i, err)
		}

		rule, err := parser.ParseRule()
		if err != nil {
			return nil, fmt.Errorf("failed to parse rule %d (%s): %w", i, ruleStr, err)
		}

		predicate, err := Compile(rule.Expr)
		if err != nil {
			return nil, fmt.Errorf("failed to compile rule %d (%s): %w", i, ruleStr, err)
		}

		filter.Rules = append(filter.Rules, CompiledRule{
			Rule:      rule,
			Predicate: predicate,
			RuleText:  ruleStr,
		})
	}

	return filter, nil
}

// Evaluate evaluates the filter against an event
// Returns: (matched bool, shouldLog bool, matchedRuleIndex int)
// If no rule matches, returns (false, true, -1) for log-by-default policy
func (f *Filter) Evaluate(event conntrack.Event) (bool, bool, int) {
	if f == nil || len(f.Rules) == 0 {
		// Log by default when no rules
		return false, true, -1
	}

	// First-match wins
	for i, compiledRule := range f.Rules {
		if compiledRule.Predicate(event) {
			shouldLog := compiledRule.Rule.Action == ActionLog
			return true, shouldLog, i
		}
	}

	// Log by default when no rule matches
	return false, true, -1
}
