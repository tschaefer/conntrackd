/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package filter

import (
	"fmt"
	"strings"
	"unicode"
)

type TokenType int

const (
	TokenEOF TokenType = iota
	TokenIdent
	TokenNumber
	TokenComma
	TokenDash
	TokenSlash
	TokenColon
	TokenDot
	TokenLParen
	TokenRParen
	TokenAnd
	TokenOr
	TokenNot
	TokenLog
	TokenDrop
	TokenEventType
	TokenProtocol
	TokenSource
	TokenDestination
	TokenAddress
	TokenNetwork
	TokenPort
	TokenOn
	TokenAny
)

type Token struct {
	Type  TokenType
	Value string
	Pos   int
}

type Lexer struct {
	input string
	pos   int
	ch    rune
}

func NewLexer(input string) *Lexer {
	l := &Lexer{input: input}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.pos >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = rune(l.input[l.pos])
	}
	l.pos++
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) readIdentifier() string {
	start := l.pos - 1
	for unicode.IsLetter(l.ch) || unicode.IsDigit(l.ch) || l.ch == '_' {
		l.readChar()
	}
	return l.input[start : l.pos-1]
}

func (l *Lexer) readNumber() string {
	start := l.pos - 1
	for unicode.IsDigit(l.ch) {
		l.readChar()
	}
	return l.input[start : l.pos-1]
}

func (l *Lexer) NextToken() (Token, error) {
	l.skipWhitespace()

	tok := Token{Pos: l.pos - 1}

	switch l.ch {
	case 0:
		tok.Type = TokenEOF
	case ',':
		tok.Type = TokenComma
		tok.Value = ","
		l.readChar()
	case '-':
		tok.Type = TokenDash
		tok.Value = "-"
		l.readChar()
	case '/':
		tok.Type = TokenSlash
		tok.Value = "/"
		l.readChar()
	case ':':
		tok.Type = TokenColon
		tok.Value = ":"
		l.readChar()
	case '.':
		tok.Type = TokenDot
		tok.Value = "."
		l.readChar()
	case '(':
		tok.Type = TokenLParen
		tok.Value = "("
		l.readChar()
	case ')':
		tok.Type = TokenRParen
		tok.Value = ")"
		l.readChar()
	case '!':
		tok.Type = TokenNot
		tok.Value = "!"
		l.readChar()
	default:
		if unicode.IsLetter(l.ch) {
			tok.Value = l.readIdentifier()
			tok.Type = l.lookupKeyword(tok.Value)
		} else if unicode.IsDigit(l.ch) {
			tok.Value = l.readNumber()
			tok.Type = TokenNumber
		} else {
			return tok, fmt.Errorf("unexpected character: %c at position %d", l.ch, l.pos-1)
		}
	}

	return tok, nil
}

func (l *Lexer) lookupKeyword(ident string) TokenType {
	keywords := map[string]TokenType{
		"and":         TokenAnd,
		"or":          TokenOr,
		"not":         TokenNot,
		"log":         TokenLog,
		"drop":        TokenDrop,
		"type":        TokenEventType,
		"protocol":    TokenProtocol,
		"source":      TokenSource,
		"src":         TokenSource,
		"destination": TokenDestination,
		"dst":         TokenDestination,
		"dest":        TokenDestination,
		"address":     TokenAddress,
		"network":     TokenNetwork,
		"port":        TokenPort,
		"on":          TokenOn,
		"any":         TokenAny,
	}

	lower := strings.ToLower(ident)
	if tokType, ok := keywords[lower]; ok {
		return tokType
	}
	return TokenIdent
}
