/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package filter

import (
	"fmt"
	"net/netip"
	"strings"
	"syscall"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/ti-mo/conntrack"
)

// Filter represents a CEL-based filter
type Filter struct {
	rules []compiledRule
}

// compiledRule represents a compiled CEL filter rule
type compiledRule struct {
	action   action
	program  cel.Program
	ruleText string
}

// action represents the action to take when a rule matches
type action int

const (
	actionLog action = iota
	actionDrop
)

func (a action) String() string {
	switch a {
	case actionLog:
		return "log"
	case actionDrop:
		return "drop"
	default:
		return "unknown"
	}
}

// NewFilter creates a new CEL-based filter from rule strings
func NewFilter(ruleStrings []string) (*Filter, error) {
	filter := &Filter{
		rules: make([]compiledRule, 0, len(ruleStrings)),
	}

	env, err := createCELEnvironment()
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	for i, ruleStr := range ruleStrings {
		act, expr, err := parseRuleString(ruleStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse rule %d (%s): %w", i, ruleStr, err)
		}

		if expr == "any" {
			expr = "true"
		}

		ast, issues := env.Compile(expr)
		if issues != nil && issues.Err() != nil {
			return nil, fmt.Errorf("failed to compile rule %d (%s): %w", i, ruleStr, issues.Err())
		}

		program, err := env.Program(ast)
		if err != nil {
			return nil, fmt.Errorf("failed to create program for rule %d (%s): %w", i, ruleStr, err)
		}

		filter.rules = append(filter.rules, compiledRule{
			action:   act,
			program:  program,
			ruleText: ruleStr,
		})
	}

	return filter, nil
}

// Evaluate evaluates the filter against an event
// Returns: (matched bool, shouldLog bool, matchedRuleIndex int)
// If no rule matches, returns (false, true, -1) for log-by-default policy
func (f *Filter) Evaluate(event conntrack.Event) (bool, bool, int) {
	if f == nil || len(f.rules) == 0 {
		return false, true, -1
	}

	ctx := createEventContext(event)

	for i, compiledRule := range f.rules {
		result, _, err := compiledRule.program.Eval(ctx)
		if err != nil {
			continue
		}

		if result == types.True {
			shouldLog := compiledRule.action == actionLog
			return true, shouldLog, i
		}
	}

	return false, true, -1
}

// createCELEnvironment creates a CEL environment with custom functions
func createCELEnvironment() (*cel.Env, error) {
	return cel.NewEnv(
		cel.Variable("event.type", cel.StringType),
		cel.Variable("protocol", cel.StringType),
		cel.Variable("source.address", cel.StringType),
		cel.Variable("destination.address", cel.StringType),
		cel.Variable("source.port", cel.IntType),
		cel.Variable("destination.port", cel.IntType),

		cel.Function("is_network",
			cel.Overload("is_network_string_string",
				[]*cel.Type{cel.StringType, cel.StringType},
				cel.BoolType,
				cel.BinaryBinding(isNetworkFunc)),
		),
		cel.Function("in_cidr",
			cel.Overload("in_cidr_string_string",
				[]*cel.Type{cel.StringType, cel.StringType},
				cel.BoolType,
				cel.BinaryBinding(inCIDRFunc)),
		),
		cel.Function("in_range",
			cel.Overload("in_range_int_int_int",
				[]*cel.Type{cel.IntType, cel.IntType, cel.IntType},
				cel.BoolType,
				cel.FunctionBinding(inRangeFunc)),
		),
	)
}

// isNetworkFunc checks if an IP address belongs to a network category
func isNetworkFunc(lhs ref.Val, rhs ref.Val) ref.Val {
	ipStr, ok := lhs.(types.String)
	if !ok {
		return types.NewErr("invalid IP address type")
	}
	network, ok := rhs.(types.String)
	if !ok {
		return types.NewErr("invalid network type")
	}

	ip, err := netip.ParseAddr(string(ipStr))
	if err != nil {
		return types.Bool(false)
	}

	isLocal := ip.IsLoopback() || ip.IsLinkLocalUnicast()
	isPrivate := ip.IsPrivate()
	isMulticast := ip.IsMulticast()
	isPublic := !isLocal && !isPrivate && !isMulticast

	switch string(network) {
	case "LOCAL":
		return types.Bool(isLocal)
	case "PRIVATE":
		return types.Bool(isPrivate)
	case "MULTICAST":
		return types.Bool(isMulticast)
	case "PUBLIC":
		return types.Bool(isPublic)
	default:
		return types.Bool(false)
	}
}

// inCIDRFunc checks if an IP address is in a CIDR range
func inCIDRFunc(lhs ref.Val, rhs ref.Val) ref.Val {
	ipStr, ok := lhs.(types.String)
	if !ok {
		return types.NewErr("invalid IP address type")
	}
	cidrStr, ok := rhs.(types.String)
	if !ok {
		return types.NewErr("invalid CIDR type")
	}

	ip, err := netip.ParseAddr(string(ipStr))
	if err != nil {
		return types.Bool(false)
	}

	prefix, err := netip.ParsePrefix(string(cidrStr))
	if err != nil {
		return types.Bool(false)
	}

	return types.Bool(prefix.Contains(ip))
}

// inRangeFunc checks if a value is in a range (inclusive)
func inRangeFunc(args ...ref.Val) ref.Val {
	if len(args) != 3 {
		return types.NewErr("in_range requires 3 arguments")
	}

	val, ok := args[0].(types.Int)
	if !ok {
		return types.NewErr("first argument must be int")
	}
	min, ok := args[1].(types.Int)
	if !ok {
		return types.NewErr("second argument must be int")
	}
	max, ok := args[2].(types.Int)
	if !ok {
		return types.NewErr("third argument must be int")
	}

	return types.Bool(val >= min && val <= max)
}

// createEventContext creates a CEL evaluation context from a conntrack event
func createEventContext(event conntrack.Event) map[string]any {
	var eventType string
	switch event.Type {
	case conntrack.EventNew:
		eventType = "NEW"
	case conntrack.EventUpdate:
		eventType = "UPDATE"
	case conntrack.EventDestroy:
		eventType = "DESTROY"
	}

	var protocol string
	switch event.Flow.TupleOrig.Proto.Protocol {
	case syscall.IPPROTO_TCP:
		protocol = "TCP"
	case syscall.IPPROTO_UDP:
		protocol = "UDP"
	}

	return map[string]any{
		"event.type":          eventType,
		"protocol":            protocol,
		"source.address":      event.Flow.TupleOrig.IP.SourceAddress.String(),
		"destination.address": event.Flow.TupleOrig.IP.DestinationAddress.String(),
		"source.port":         int64(event.Flow.TupleOrig.Proto.SourcePort),
		"destination.port":    int64(event.Flow.TupleOrig.Proto.DestinationPort),
	}
}

// parseRuleString parses a rule string to extract action and CEL expression
// Format: "log <expression>" or "drop <expression>"
func parseRuleString(ruleStr string) (action, string, error) {
	ruleStr = strings.TrimSpace(ruleStr)

	if strings.HasPrefix(strings.ToLower(ruleStr), "log ") {
		expr := strings.TrimSpace(ruleStr[4:])
		if expr == "" {
			return 0, "", fmt.Errorf("missing expression after 'log'")
		}
		return actionLog, expr, nil
	}

	if strings.HasPrefix(strings.ToLower(ruleStr), "drop ") {
		expr := strings.TrimSpace(ruleStr[5:])
		if expr == "" {
			return 0, "", fmt.Errorf("missing expression after 'drop'")
		}
		return actionDrop, expr, nil
	}

	return 0, "", fmt.Errorf("rule must start with 'log' or 'drop'")
}
