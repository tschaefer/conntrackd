/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package filter

// Action represents the action to take when a rule matches
type Action int

const (
	ActionLog Action = iota
	ActionDrop
)

func (a Action) String() string {
	switch a {
	case ActionLog:
		return "log"
	case ActionDrop:
		return "drop"
	default:
		return "unknown"
	}
}

// ExprNode represents a node in the expression AST
type ExprNode interface {
	isExprNode()
}

// BinaryExpr represents a binary expression (AND, OR)
type BinaryExpr struct {
	Op    BinaryOp
	Left  ExprNode
	Right ExprNode
}

func (BinaryExpr) isExprNode() {}

type BinaryOp int

const (
	OpAnd BinaryOp = iota
	OpOr
)

// UnaryExpr represents a unary expression (NOT)
type UnaryExpr struct {
	Op   UnaryOp
	Expr ExprNode
}

func (UnaryExpr) isExprNode() {}

type UnaryOp int

const (
	OpNot UnaryOp = iota
)

// Predicate represents a base predicate
type Predicate interface {
	ExprNode
	isPredicate()
}

// TypePredicate matches event types
type TypePredicate struct {
	Types []string // NEW, UPDATE, DESTROY
}

func (TypePredicate) isExprNode()  {}
func (TypePredicate) isPredicate() {}

// ProtocolPredicate matches protocols
type ProtocolPredicate struct {
	Protocols []string // TCP, UDP
}

func (ProtocolPredicate) isExprNode()  {}
func (ProtocolPredicate) isPredicate() {}

// NetworkPredicate matches network types
type NetworkPredicate struct {
	Direction string   // source, destination
	Networks  []string // LOCAL, PRIVATE, PUBLIC, MULTICAST
}

func (NetworkPredicate) isExprNode()  {}
func (NetworkPredicate) isPredicate() {}

// AddressPredicate matches IP addresses or CIDR ranges
type AddressPredicate struct {
	Direction string   // source, destination
	Addresses []string // IP addresses or CIDR
	Ports     []uint16 // optional ports
}

func (AddressPredicate) isExprNode()  {}
func (AddressPredicate) isPredicate() {}

// PortPredicate matches ports
type PortPredicate struct {
	Direction string   // source, destination
	Ports     []uint16 // port numbers or ranges
}

func (PortPredicate) isExprNode()  {}
func (PortPredicate) isPredicate() {}

// AnyPredicate matches any event (catch-all)
type AnyPredicate struct{}

func (AnyPredicate) isExprNode()  {}
func (AnyPredicate) isPredicate() {}

// Rule represents a complete filter rule
type Rule struct {
	Action Action
	Expr   ExprNode
}
