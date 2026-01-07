/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package filter

import (
	"fmt"
	"strconv"
	"strings"
)

type Parser struct {
	lexer   *Lexer
	current Token
	peek    Token
}

func NewParser(input string) (*Parser, error) {
	p := &Parser{lexer: NewLexer(input)}
	// Read two tokens to initialize current and peek
	if err := p.nextToken(); err != nil {
		return nil, err
	}
	if err := p.nextToken(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Parser) nextToken() error {
	p.current = p.peek
	tok, err := p.lexer.NextToken()
	if err != nil {
		return err
	}
	p.peek = tok
	return nil
}

func (p *Parser) expect(tokType TokenType) error {
	if p.current.Type != tokType {
		return fmt.Errorf("expected token %v, got %v (%s) at position %d",
			tokType, p.current.Type, p.current.Value, p.current.Pos)
	}
	return p.nextToken()
}

// ParseRule parses a complete rule: action expression
func (p *Parser) ParseRule() (*Rule, error) {
	rule := &Rule{}

	// Parse action
	switch p.current.Type {
	case TokenLog:
		rule.Action = ActionLog
		if err := p.nextToken(); err != nil {
			return nil, err
		}
	case TokenDrop:
		rule.Action = ActionDrop
		if err := p.nextToken(); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("expected 'log' or 'drop', got '%s' at position %d",
			p.current.Value, p.current.Pos)
	}

	// Parse expression
	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	rule.Expr = expr

	// Expect EOF
	if p.current.Type != TokenEOF {
		return nil, fmt.Errorf("unexpected token after rule: %s at position %d",
			p.current.Value, p.current.Pos)
	}

	return rule, nil
}

// parseExpression parses: orExpr
func (p *Parser) parseExpression() (ExprNode, error) {
	return p.parseOrExpr()
}

// parseOrExpr parses: andExpr { ("," | "or") andExpr }
func (p *Parser) parseOrExpr() (ExprNode, error) {
	left, err := p.parseAndExpr()
	if err != nil {
		return nil, err
	}

	for p.current.Type == TokenOr || p.current.Type == TokenComma {
		if err := p.nextToken(); err != nil {
			return nil, err
		}
		right, err := p.parseAndExpr()
		if err != nil {
			return nil, err
		}
		left = BinaryExpr{Op: OpOr, Left: left, Right: right}
	}

	return left, nil
}

// parseAndExpr parses: notExpr { "and" notExpr }
func (p *Parser) parseAndExpr() (ExprNode, error) {
	left, err := p.parseNotExpr()
	if err != nil {
		return nil, err
	}

	for p.current.Type == TokenAnd {
		if err := p.nextToken(); err != nil {
			return nil, err
		}
		right, err := p.parseNotExpr()
		if err != nil {
			return nil, err
		}
		left = BinaryExpr{Op: OpAnd, Left: left, Right: right}
	}

	return left, nil
}

// parseNotExpr parses: [ "not" | "!" ] primary
func (p *Parser) parseNotExpr() (ExprNode, error) {
	if p.current.Type == TokenNot {
		if err := p.nextToken(); err != nil {
			return nil, err
		}
		expr, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		return UnaryExpr{Op: OpNot, Expr: expr}, nil
	}
	return p.parsePrimary()
}

// parsePrimary parses: predicate | "(" expression ")"
func (p *Parser) parsePrimary() (ExprNode, error) {
	if p.current.Type == TokenLParen {
		if err := p.nextToken(); err != nil {
			return nil, err
		}
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if err := p.expect(TokenRParen); err != nil {
			return nil, err
		}
		return expr, nil
	}
	return p.parsePredicate()
}

// parsePredicate parses various predicate types
func (p *Parser) parsePredicate() (ExprNode, error) {
	switch p.current.Type {
	case TokenEventType:
		return p.parseTypePredicate()
	case TokenProtocol:
		return p.parseProtocolPredicate()
	case TokenSource, TokenDestination:
		return p.parseDirectionalPredicate()
	case TokenOn:
		return p.parsePortPredicate()
	case TokenAny:
		// Parse "any" - matches everything
		if err := p.nextToken(); err != nil {
			return nil, err
		}
		return AnyPredicate{}, nil
	default:
		return nil, fmt.Errorf("expected predicate keyword, got '%s' at position %d",
			p.current.Value, p.current.Pos)
	}
}

// parseTypePredicate parses: "type" IDENT_LIST
func (p *Parser) parseTypePredicate() (ExprNode, error) {
	if err := p.nextToken(); err != nil {
		return nil, err
	}

	types, err := p.parseIdentList()
	if err != nil {
		return nil, err
	}

	// Normalize and validate type names
	validTypes := map[string]bool{
		"NEW":     true,
		"UPDATE":  true,
		"DESTROY": true,
	}
	normalized := make([]string, len(types))
	for i, t := range types {
		normalized[i] = strings.ToUpper(t)
		if !validTypes[normalized[i]] {
			return nil, fmt.Errorf("invalid event type '%s' at position %d, valid types are: NEW, UPDATE, DESTROY", t, p.current.Pos)
		}
	}

	return TypePredicate{Types: normalized}, nil
}

// parseProtocolPredicate parses: "protocol" IDENT_LIST
func (p *Parser) parseProtocolPredicate() (ExprNode, error) {
	if err := p.nextToken(); err != nil {
		return nil, err
	}

	protocols, err := p.parseIdentList()
	if err != nil {
		return nil, err
	}

	// Normalize and validate protocol names
	validProtocols := map[string]bool{
		"TCP": true,
		"UDP": true,
	}
	normalized := make([]string, len(protocols))
	for i, proto := range protocols {
		normalized[i] = strings.ToUpper(proto)
		if !validProtocols[normalized[i]] {
			return nil, fmt.Errorf("invalid protocol '%s' at position %d, valid protocols are: TCP, UDP", proto, p.current.Pos)
		}
	}

	return ProtocolPredicate{Protocols: normalized}, nil
}

// parseDirectionalPredicate handles source/destination address, network, or port
func (p *Parser) parseDirectionalPredicate() (ExprNode, error) {
	direction := p.current.Value
	if err := p.nextToken(); err != nil {
		return nil, err
	}

	switch p.current.Type {
	case TokenNetwork:
		return p.parseNetworkPredicate(direction)
	case TokenAddress:
		return p.parseAddressPredicate(direction)
	case TokenPort:
		return p.parsePortPredicateWithDirection(direction)
	default:
		return nil, fmt.Errorf("expected 'network', 'address', or 'port' after '%s', got '%s' at position %d",
			direction, p.current.Value, p.current.Pos)
	}
}

// parseNetworkPredicate parses: direction "network" IDENT
func (p *Parser) parseNetworkPredicate(direction string) (ExprNode, error) {
	if err := p.nextToken(); err != nil {
		return nil, err
	}

	networks, err := p.parseIdentList()
	if err != nil {
		return nil, err
	}

	// Normalize and validate network names
	validNetworks := map[string]bool{
		"LOCAL":     true,
		"PRIVATE":   true,
		"PUBLIC":    true,
		"MULTICAST": true,
	}
	normalized := make([]string, len(networks))
	for i, net := range networks {
		normalized[i] = strings.ToUpper(net)
		if !validNetworks[normalized[i]] {
			return nil, fmt.Errorf("invalid network type '%s' at position %d, valid networks are: LOCAL, PRIVATE, PUBLIC, MULTICAST", net, p.current.Pos)
		}
	}

	return NetworkPredicate{Direction: direction, Networks: normalized}, nil
}

// parseAddressPredicate parses: direction "address" (IP | CIDR) ["on" "port" PORT_SPEC]
func (p *Parser) parseAddressPredicate(direction string) (ExprNode, error) {
	if err := p.nextToken(); err != nil {
		return nil, err
	}

	addresses, err := p.parseAddressList()
	if err != nil {
		return nil, err
	}

	var ports []uint16
	// Check for optional "on port"
	if p.current.Type == TokenOn {
		if err := p.nextToken(); err != nil {
			return nil, err
		}
		if err := p.expect(TokenPort); err != nil {
			return nil, err
		}
		ports, err = p.parsePortSpec()
		if err != nil {
			return nil, err
		}
	}

	return AddressPredicate{Direction: direction, Addresses: addresses, Ports: ports}, nil
}

// parsePortPredicate parses: "on" "port" PORT_SPEC (no direction)
func (p *Parser) parsePortPredicate() (ExprNode, error) {
	if err := p.nextToken(); err != nil {
		return nil, err
	}
	if err := p.expect(TokenPort); err != nil {
		return nil, err
	}

	ports, err := p.parsePortSpec()
	if err != nil {
		return nil, err
	}

	// "on port" without direction means both source and destination
	return PortPredicate{Direction: "both", Ports: ports}, nil
}

// parsePortPredicateWithDirection parses: direction "port" PORT_SPEC
func (p *Parser) parsePortPredicateWithDirection(direction string) (ExprNode, error) {
	if err := p.nextToken(); err != nil {
		return nil, err
	}

	ports, err := p.parsePortSpec()
	if err != nil {
		return nil, err
	}

	return PortPredicate{Direction: direction, Ports: ports}, nil
}

// parseIdentList parses: IDENT { "," IDENT }
func (p *Parser) parseIdentList() ([]string, error) {
	var idents []string

	if p.current.Type != TokenIdent {
		return nil, fmt.Errorf("expected identifier, got '%s' at position %d",
			p.current.Value, p.current.Pos)
	}

	idents = append(idents, p.current.Value)
	if err := p.nextToken(); err != nil {
		return nil, err
	}

	for p.current.Type == TokenComma {
		if err := p.nextToken(); err != nil {
			return nil, err
		}
		if p.current.Type != TokenIdent {
			return nil, fmt.Errorf("expected identifier after comma, got '%s' at position %d",
				p.current.Value, p.current.Pos)
		}
		idents = append(idents, p.current.Value)
		if err := p.nextToken(); err != nil {
			return nil, err
		}
	}

	return idents, nil
}

// parseAddressList parses IP addresses or CIDR ranges
func (p *Parser) parseAddressList() ([]string, error) {
	var addresses []string

	addr, err := p.parseAddress()
	if err != nil {
		return nil, err
	}
	addresses = append(addresses, addr)

	for p.current.Type == TokenComma {
		if err := p.nextToken(); err != nil {
			return nil, err
		}
		addr, err := p.parseAddress()
		if err != nil {
			return nil, err
		}
		addresses = append(addresses, addr)
	}

	return addresses, nil
}

// parseAddress parses an IP address or CIDR range
func (p *Parser) parseAddress() (string, error) {
	var parts []string

	// Parse IPv4, IPv6, or CIDR
	for {
		switch p.current.Type {
		case TokenNumber, TokenIdent:
			parts = append(parts, p.current.Value)
			if err := p.nextToken(); err != nil {
				return "", err
			}
		case TokenDot, TokenColon, TokenSlash:
			parts = append(parts, p.current.Value)
			if err := p.nextToken(); err != nil {
				return "", err
			}
		default:
			goto done
		}
	}
done:

	if len(parts) == 0 {
		return "", fmt.Errorf("expected IP address at position %d", p.current.Pos)
	}

	return strings.Join(parts, ""), nil
}

// parsePortSpec parses: NUMBER | NUMBER "-" NUMBER | NUMBER { "," NUMBER }
func (p *Parser) parsePortSpec() ([]uint16, error) {
	var ports []uint16

	if p.current.Type != TokenNumber {
		return nil, fmt.Errorf("expected port number, got '%s' at position %d",
			p.current.Value, p.current.Pos)
	}

	port, err := strconv.ParseUint(p.current.Value, 10, 16)
	if err != nil {
		return nil, fmt.Errorf("invalid port number '%s' at position %d",
			p.current.Value, p.current.Pos)
	}
	ports = append(ports, uint16(port))

	if err := p.nextToken(); err != nil {
		return nil, err
	}

	// Check for range (e.g., 80-90)
	if p.current.Type == TokenDash {
		if err := p.nextToken(); err != nil {
			return nil, err
		}
		if p.current.Type != TokenNumber {
			return nil, fmt.Errorf("expected port number after '-', got '%s' at position %d",
				p.current.Value, p.current.Pos)
		}
		endPort, err := strconv.ParseUint(p.current.Value, 10, 16)
		if err != nil {
			return nil, fmt.Errorf("invalid port number '%s' at position %d",
				p.current.Value, p.current.Pos)
		}
		// Expand range
		for p := ports[0] + 1; p <= uint16(endPort); p++ {
			ports = append(ports, p)
		}
		if err := p.nextToken(); err != nil {
			return nil, err
		}
		return ports, nil
	}

	// Check for comma-separated list
	for p.current.Type == TokenComma {
		if err := p.nextToken(); err != nil {
			return nil, err
		}
		if p.current.Type != TokenNumber {
			return nil, fmt.Errorf("expected port number after comma, got '%s' at position %d",
				p.current.Value, p.current.Pos)
		}
		port, err := strconv.ParseUint(p.current.Value, 10, 16)
		if err != nil {
			return nil, fmt.Errorf("invalid port number '%s' at position %d",
				p.current.Value, p.current.Pos)
		}
		ports = append(ports, uint16(port))
		if err := p.nextToken(); err != nil {
			return nil, err
		}
	}

	return ports, nil
}
