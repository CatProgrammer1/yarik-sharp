package main

import (
	//"fmt"

	"strconv"
	"strings"
	"unicode"
)

const (
	boolTrue  = "true"
	boolFalse = "false"

	selfKeyword = "this"
)

var (
	tokenTypes = map[string]string{
		//keywords
		"yar":      "var",
		"if":       "ifstmt",
		"else":     "else",
		"func":     "func",
		"while":    "wlloop",
		"foreach":  "forloop",
		"break":    "break",
		"continue": "continue",
		"return":   "return",
		"import":   "import",
		"new":      "newstruct",
		"struct":   "struct",

		"#bits": "valbitcount",

		boolTrue:  "bool",
		boolFalse: "bool",

		"void": "nil",

		//operators
		"=": "assign",
		"+": "add",
		"-": "sub",
		"/": "div",
		"*": "mul",
		"|": "bitor",
		"&": "getptr",

		"->": "indexstruct",

		"(": "openbracket",
		")": "closebracket",
		"{": "openbrace",
		"}": "closebrace",
		"[": "opensqbrac",
		"]": "closesqbrac",
		",": "comma",
		":": "colon",

		//tags
		"#": "fieldctype",

		//comment
		"/*": "cmtopen",
		"*/": "cmtclose",

		//boolean operators
		"==": "equals",
		"!=": "notequals",
		">":  "greater",
		"<":  "less",
		">=": "greatereq",
		"<=": "lesseq",

		//stmt operators
		"&&": "and",
		"||": "or",
	}
	metacharsValue = map[string]string{
		"n":  "\n",
		"t":  "\t",
		"r":  "\r",
		"\\": "\\",
		"b":  "\b",
		"\"": "\"",
		"'":  "'",
		"0":  "\x00",
	}
)

const (
	ignore      = " \n\t\r"
	digits      = "0123456789"
	stringChars = "'\"`"
	metachars   = "nrt\\b\"'0"
)

func tonumber(str string) float64 {
	n, err := strconv.ParseFloat(str, 64)
	handle(err)
	return n
}

type Lexer struct {
	Source          string
	SourceChar      []rune
	CurrentPosition int
	CurrentColumn   int
	CurrentLine     int
	CurrentChar     rune
}

func NewLexer(src string) *Lexer {
	return &Lexer{
		Source:     src,
		SourceChar: []rune(src),
	}
}

type Token struct {
	Value          any
	Type           string
	Position, Line int
}

func NewToken(Value any, Type string, Column, Line int) Token {
	return Token{Value, Type, Column, Line}
}

func (lexer *Lexer) LoadSourceChars() {
	lexer.SourceChar = []rune(lexer.Source)
}

func (lexer *Lexer) Char() rune {
	return lexer.CurrentChar
}

func (lexer *Lexer) Str() string {
	return string(lexer.CurrentChar)
}

func (lexer *Lexer) Next() {
	lexer.CurrentPosition++
	lexer.CurrentColumn++
	if lexer.CurrentPosition+1 > len(lexer.SourceChar) {
		lexer.CurrentPosition = -1
		return
	}
	lexer.CurrentChar = lexer.SourceChar[lexer.CurrentPosition]
}

func (lexer *Lexer) NextTimes(times int) {
	for i := 0; i < times; i++ {
		if lexer.CurrentPosition == -1 {
			break
		}
		lexer.Next()
	}
}

func (lexer *Lexer) PeekNext() rune {
	nextPos := lexer.CurrentPosition + 1
	if nextPos+1 > len(lexer.SourceChar) {
		return 0
	}
	return lexer.SourceChar[nextPos]
}

func (lexer *Lexer) PeekPrev() rune {
	prevPos := lexer.CurrentPosition - 1
	if prevPos+1 > 0 {
		return 0
	}
	return lexer.SourceChar[prevPos]
}

func (lexer *Lexer) GetTokens() []Token {
	lexer.CurrentPosition = -1
	lexer.CurrentLine = 1
	lexer.CurrentColumn = 0
	lexer.Next()

	tokens := []Token{}

	for lexer.CurrentPosition >= 0 {
		charStr := lexer.Str()
		if strings.Contains(ignore, charStr) {
			if charStr == "\n" {
				lexer.CurrentLine++
				lexer.CurrentColumn = 0
			}
			lexer.Next()
			continue
		}
		switch {
		case strings.Contains(stringChars, charStr):
			tokens = append(tokens, lexer.GetString(charStr))
		case strings.Contains(digits, charStr):
			tokens = append(tokens, lexer.GetNumber())
		default:

			found := false

			var tokenStr string

			for k := range tokenTypes {
				if !strings.Contains(k, charStr) {
					continue
				}

				endPos := lexer.CurrentPosition + len(k)

				if endPos >= len(lexer.SourceChar)+1 {
					continue
				}
				fullk := string(lexer.SourceChar[lexer.CurrentPosition:endPos])
				if fullk != k || len(fullk) <= len(tokenStr) {
					continue
				}
				if onlyLetters(fullk) && !unicode.IsSpace(lexer.SourceChar[endPos]) {
					continue
				}

				found = true
				tokenStr = k
			}

			if !found {
				ident := lexer.GetIdentifier()
				if ident.Value == nil {
					throw("Kapec", lexer.CurrentColumn, lexer.CurrentLine)
				}

				if ident.Value == boolTrue {
					tokens = append(tokens, NewToken(true, "bool", lexer.CurrentColumn, lexer.CurrentLine))
					break
				} else if ident.Value == boolFalse {
					tokens = append(tokens, NewToken(false, "bool", lexer.CurrentColumn, lexer.CurrentLine))
					break
				}

				tokens = append(tokens, ident)
			} else {
				lexer.NextTimes(len(tokenStr))

				if tokenStr == boolTrue {
					tokens = append(tokens, NewToken(true, "bool", lexer.CurrentColumn, lexer.CurrentLine))
					break
				} else if tokenStr == boolFalse {
					tokens = append(tokens, NewToken(false, "bool", lexer.CurrentColumn, lexer.CurrentLine))
					break
				}

				tokens = append(tokens, NewToken(tokenStr, tokenTypes[tokenStr], lexer.CurrentColumn, lexer.CurrentLine))
			}
		}
	}
	tokens = append(tokens, NewToken("EOF", "EOF", lexer.CurrentColumn, lexer.CurrentLine))
	return tokens
}

func onlyLetters(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

func (lexer *Lexer) GetNumber() Token {
	var number string
	dots := 0

	for lexer.CurrentPosition >= 0 {
		charStr := lexer.Str()
		if !strings.Contains(digits+".", charStr) {
			break
		}
		if charStr == "." {
			dots++
		}
		if dots > 1 {
			break
		}
		number += charStr

		lexer.Next()
	}

	return NewToken(tonumber(number), "number", lexer.CurrentPosition, lexer.CurrentLine)
}

func (lexer *Lexer) GetIdentifier() Token {
	var ident string
	first := true

	for lexer.CurrentPosition >= 0 {
		char := lexer.Char()
		if !unicode.IsLetter(char) && string(char) != "_" && !(!first && unicode.IsDigit(char)) {
			break
		}
		ident += string(char)
		lexer.Next()
		if first {
			first = false
		}
	}

	return NewToken(ident, "ident", lexer.CurrentColumn, lexer.CurrentLine)
}

func (lexer *Lexer) GetString(startChar string) Token {
	lexer.Next()
	var str string

	for {
		if lexer.CurrentPosition < 0 {
			throw("Lox", lexer.CurrentColumn, lexer.CurrentLine)
			break
		}
		charStr := lexer.Str()
		if charStr == startChar {
			lexer.Next()
			break
		}
		if charStr == "\\" {
			nextChar := string(lexer.PeekNext())
			if strings.Contains(metachars, nextChar) {
				str += metacharsValue[nextChar]
				lexer.NextTimes(2)
				continue
			}
		}
		str += charStr
		lexer.Next()
	}

	return NewToken(str, "string", lexer.CurrentColumn, lexer.CurrentLine)
}
