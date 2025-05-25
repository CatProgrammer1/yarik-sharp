package main

import (
	"fmt"
	"slices"
	"strings"
)

var (
	tokenTypesExpects = map[string][]string{
		"number,ident,string,bool,openbracket": {"opensqbrac", "add", "sub", "div", "mul", "pow",
			"equals", "notequals", "greater", "less", "greatereq", "lesseq", "and", "or"},

		"add,sub,div,mul,pow,equals,notequals,greater,less,greatereq,lesseq,and,or": {"number", "ident", "string", "bool", "openbracket"},
		"opensqbrac": {"opensqbrac"},
	}
)

type Parser struct {
	CurrentToken    Token
	CurrentPosition int
	Tokens          []Token
	Expected        []string
	Unexpected      []string
}

func NewParser(tokens []Token) *Parser {
	return &Parser{
		Tokens: tokens,
	}
}

func (parser *Parser) Expect(tokenTypes ...string) {
	parser.Expected = tokenTypes
}

func (parser *Parser) Unexpect(tokenTypes ...string) {
	parser.Unexpected = tokenTypes
}

func (parser *Parser) Next(expectedTokenTypes ...string) {
	parser.Expected = expectedTokenTypes
	parser.CurrentPosition++
	if parser.CurrentPosition+1 > len(parser.Tokens) {
		parser.CurrentPosition = -1
		return
	}

	currentToken := parser.Tokens[parser.CurrentPosition]
	if len(parser.Expected) > 0 && !slices.Contains(parser.Expected, currentToken.Type) {
		panic(fmt.Sprintf("Expected '%s' got '%s'.", parser.Expected, currentToken.Type))
	}
	if len(parser.Unexpected) > 0 && slices.Contains(parser.Unexpected, currentToken.Type) {
		panic(fmt.Sprintf("Invalid token '%s'.", currentToken.Type))
	}

	parser.Unexpected = []string{}
	parser.CurrentToken = currentToken
}

func (parser *Parser) IsCurrentToken(tokenTypes ...string) bool {
	if parser.CurrentPosition < 0 {
		return false
	}
	return slices.Contains(tokenTypes, parser.CurrentToken.Type)
}

func (parser *Parser) NextTimes(times int) {
	for i := 0; i < times; i++ {
		if parser.CurrentPosition == -1 {
			break
		}
		parser.Next()
	}
}

func (parser *Parser) PeekNext() Token {
	nextPos := parser.CurrentPosition + 1
	if nextPos+1 > len(parser.Tokens) {
		return Token{}
	}
	return parser.Tokens[nextPos]
}

func (parser *Parser) PeekPrev() Token {
	prevPos := parser.CurrentPosition - 1
	if prevPos-1 < 0 {
		return Token{}
	}
	return parser.Tokens[prevPos]
}

func getLastRightOperand(binOpNode *BinOpNode) Node {
	switch rOp := binOpNode.R.(type) {
	case *BinOpNode:
		return getLastRightOperand(rOp)
	case *IdentNode, *NumNode, *BoolNode, *StrNode:
		return rOp
	}
	return nil
}

func setLastRightOperand(binOpNode *BinOpNode, value Node) {
	switch rOp := binOpNode.R.(type) {
	case *BinOpNode:
		setLastRightOperand(rOp, value)
	case *IdentNode, *NumNode, *BoolNode, *StrNode, nil:
		binOpNode.R = value
	}
}

func getLastNode(nodes []Node) Node {
	if len(nodes) == 0 {
		return nil
	}
	return nodes[len(nodes)-1]
}

func newDataTypeNode(token Token) Node {
	x, y := token.Position, token.Line

	switch token.Type {
	case "number":
		return &NumNode{
			token.Value.(float64),
			x, y,
		}
	case "string":
		return &StrNode{
			token.Value.(string),
			x, y,
		}
	case "bool":
		return &BoolNode{
			token.Value.(bool),
			x, y,
		}
	case "ident":
		return &IdentNode{
			token.Value.(string),
			x, y,
		}
	}
	return nil
}

func appendDataType(node Node, nodes []Node) []Node {
	lastNode := getLastNode(nodes)

	switch lastNode := lastNode.(type) {
	case *BinOpNode:
		lastROp := getLastRightOperand(lastNode)
		if lastROp == nil {
			setLastRightOperand(lastNode, node)
			return nodes
		}
		nodes = append(nodes, node)
	default:
		fmt.Println("Added")
		nodes = append(nodes, node)
		fmt.Println(nodes)
	}
	return nodes
}

func replaceLastNodeWith(nodes []Node, newNode Node) []Node {
	if len(nodes) == 0 {
		nodes = append(nodes, newNode)
		return nodes
	}
	nodes[len(nodes)-1] = newNode
	return nodes
}

func (parser *Parser) Parse(nodes []Node) []Node {
	if parser.CurrentPosition < 0 {
		return nil
	}
	currentToken := parser.CurrentToken
	if currentToken.Type == "EOF" {
		parser.Next()
		return nodes
	}
	x, y := currentToken.Position, currentToken.Line

	switch currentToken.Type {
	case "ifstmt":
		ifStmt := parser.ParseIfStmt()
		ifStmt.X, ifStmt.Y = x, y

		nodes = append(nodes, ifStmt)
		return nodes
	case "var":
		variable := parser.ParseVariable()
		variable.X, variable.Y = x, y

		nodes = append(nodes, variable)
		return nodes
	case "func":
		function := parser.ParseFuncDecl()
		function.X, function.Y = x, y

		nodes = append(nodes, function)
		return nodes
	case "assign":
		lastNode := getLastNode(nodes)
		fmt.Println(lastNode.(*GetElementNode))
		if lastNode != nil {
			switch lastNode := lastNode.(type) {
			case *IdentNode:
				parser.Next()

				node := &SetVar{
					lastNode,
					parser.ParseValue(),
					x, y,
				}

				return replaceLastNodeWith(nodes, node)
			case *GetElementNode:
				parser.Next()

				node := &SetElem{
					lastNode,
					parser.ParseValue(),
					x, y,
				}

				return replaceLastNodeWith(nodes, node)
			}
		}
	case "add", "sub", "div", "mul", "pow",
		"equals", "notequals", "greater", "less", "greatereq", "lesseq",
		"and", "or":
		lastNode := getLastNode(nodes)
		if lastNode == nil {
			fmt.Println(nodes)
			panic(fmt.Sprintf("Expected left operand for '%s' binary operation got nothing", currentToken.Type))
		} else {

			binOpNode := &BinOpNode{
				operator: currentToken.Type,
				X:        x,
				Y:        y,
			}
			switch node := lastNode.(type) {
			case *BinOpNode:
				switch node.operator {
				case "mul", "div", "pow":
					binOpNode.L = node
					nodes = replaceLastNodeWith(nodes, binOpNode)
				default:
					if currentToken.Type == "and" || currentToken.Type == "or" {
						binOpNode.L = node
						nodes = replaceLastNodeWith(nodes, binOpNode)
					} else {
						binOpNode.L = getLastRightOperand(node)
						setLastRightOperand(node, binOpNode)
					}
				}
			default:
				binOpNode.L = node
				nodes = replaceLastNodeWith(nodes, binOpNode)
			}
		}
		parser.Next()
		return nodes
	case "openbracket":
		lastNode := getLastNode(nodes)

		switch lastNode := lastNode.(type) {
		case *BinOpNode:
			R := getLastRightOperand(lastNode)

			if R != nil {
				funcCall := parser.ParseFuncCall()
				funcCall.Func = R
				setLastRightOperand(lastNode, funcCall)
				return nodes
			}

		case *IdentNode:
			funcCall := parser.ParseFuncCall()
			funcCall.Func = lastNode
			funcCall.X, funcCall.Y = x, y
			return replaceLastNodeWith(nodes, funcCall)
		}
		brackets := parser.ParseBrackets()
		brackets.X, brackets.Y = x, y
		nodes = appendDataType(brackets, nodes)

		return nodes
	case "opensqbrac":
		lastNode := getLastNode(nodes)

		switch lastNode := lastNode.(type) {
		case *BinOpNode:
			R := getLastRightOperand(lastNode)

			if R != nil {
				setLastRightOperand(lastNode, &GetElementNode{
					[]Node{R},
					parser.ParseKey(),
					x, y,
				})
				return nodes
			}

			nodes = appendDataType(parser.ParseMap(), nodes)
			return nodes
		case nil:
			nodes = appendDataType(parser.ParseMap(), nodes)

			return nodes
		default:
			//fmt.Println("SIGMA BOY ULTRA", lastNode.(*VarDec))
			nodes = replaceLastNodeWith(nodes, &GetElementNode{
				[]Node{lastNode},
				parser.ParseKey(),
				x, y,
			})

			return nodes
		}
	case "ident":
		nodes = appendDataType(newDataTypeNode(currentToken), nodes)
		parser.Next()

		return nodes
	case "number", "string", "bool":
		nodes = appendDataType(newDataTypeNode(currentToken), nodes)
		parser.Next()
		fmt.Println(parser.CurrentToken)
		return nodes
	}

	//fmt.Println(currentToken)
	panic(fmt.Sprintf("Invalid token '%s'.\nyks-%d:%d", currentToken.Type, currentToken.Line, currentToken.Position))
}

func (parser *Parser) ParseIfStmt() *IfStmt {
	varDec := &IfStmt{}
	condition := []Node{}

STMTPAR:
	for parser.CurrentPosition >= 0 {
		token := parser.CurrentToken

		switch token.Type {
		case "ifstmt":
			parser.Next()
		case "openbrace":
			varDec.Body = parser.ParseBody()
			break STMTPAR
		default:
			condition = parser.Parse(condition)
		}
	}
	varDec.Condition = condition

	return varDec
}

func (parser *Parser) ParseKey() []Node {
	var key []Node

	mainToken := parser.CurrentToken

KEYPAR:
	for parser.CurrentPosition >= 0 {
		token := parser.CurrentToken

		switch token.Type {
		case "opensqbrac":
			parser.Next()
		case "closesqbrac":
			parser.Next()
			break KEYPAR
		default:
			key = parser.Parse(key)
		}
	}

	if len(key) == 0 {
		throw("Key cannot be empty", 0, mainToken.Position, mainToken.Line)
	}
	return key
}

func (parser *Parser) ParseMap() *MapNode {
	mapNode := &MapNode{}

	var currentKey, currentValue []Node

MAPPAR:
	for parser.CurrentPosition >= 0 {
		token := parser.CurrentToken

		fmt.Println("Body token:", token)
		switch token.Type {
		case "colon":
			parser.Unexpect("comma", "closesqbrac")
			parser.Next()
			currentValue = parser.Parse(currentValue)
		case "comma":
			fmt.Println("ASDDADS", currentKey[0])
			if len(currentKey) > 1 || len(currentKey) == 0 {
				throw("Key has more than one value or empty.", token.Position, token.Line)
			}
			fmt.Println("AWEEEEEEEE", currentValue[0])
			if len(currentValue) > 1 || len(currentValue) == 0 {
				throw("Element has more than one value or empty.", token.Position, token.Line)
			}
			mapNode.Map = append(mapNode.Map, &Element{
				Key:   currentKey,
				Value: currentValue,
			})
			currentKey = []Node{}
			parser.Unexpect("colon")
			parser.Next()
		case "opensqbrac":
			parser.Unexpect("comma", "colon")
			parser.Next()
		case "closesqbrac":
			parser.Next()
			break MAPPAR
		default:
			currentKey = parser.Parse(currentKey)
		}
	}

	return mapNode
}

func (parser *Parser) ParseBody() []Node {
	body := []Node{}

BODYPAR:
	for parser.CurrentPosition >= 0 {
		token := parser.CurrentToken

		fmt.Println("Body token:", token)
		switch token.Type {
		case "openbrace":
			parser.Next()
		case "closebrace":
			parser.Next()
			break BODYPAR
		default:
			body = parser.Parse(body)
		}
	}

	return body
}

func (parser *Parser) ParseVariable() *VarDec {
	varDec := &VarDec{}

VARPAR:
	for parser.CurrentPosition >= 0 {
		token := parser.CurrentToken
		x, y := token.Position, token.Line

		switch token.Type {
		case "var":
			parser.Next("ident")
		case "ident":
			varDec.Identifier = IdentNode{token.Value.(string), x, y}
			parser.Next("assign")
		case "assign":
			parser.Next()
			varDec.Value = parser.ParseValue()
			fmt.Println("Ponpon")
			break VARPAR
		}
	}

	return varDec
}

func (parser *Parser) ParseDeclArgs() []IdentNode {
	args := []IdentNode{}

ARGSPAR:
	for parser.CurrentPosition >= 0 {
		token := parser.CurrentToken
		x, y := token.Position, token.Line

		switch token.Type {
		case "comma":
			parser.Next("closebracket", "ident")
		case "ident":
			args = append(args, IdentNode{token.Value.(string), x, y})
			parser.Next("closebracket", "comma")
		case "closebracket":
			break ARGSPAR
		}
	}

	return args
}

func (parser *Parser) ParseFuncDecl() *FuncDec {
	funcDec := &FuncDec{}

FUNCPAR:
	for parser.CurrentPosition >= 0 {
		token := parser.CurrentToken
		x, y := token.Position, token.Line

		switch token.Type {
		case "func":
			parser.Next("ident", "openbracket")
		case "ident":
			funcDec.Identifier = IdentNode{token.Value.(string), x, y}
			parser.Next("openbracket")
		case "openbracket":
			parser.Next("closebracket", "ident")
			funcDec.Arguments = parser.ParseDeclArgs()
		case "closebracket":
			parser.Next("openbrace")
			funcDec.Body = parser.ParseBody()
			break FUNCPAR
		}
	}

	return funcDec
}

func (parser *Parser) ParseCallArgs() []Node {
	args := []Node{}

ARGSPAR:
	for parser.CurrentPosition >= 0 {
		token := parser.CurrentToken
		fmt.Println("Args:", token)
		switch token.Type {
		case "comma":
			fmt.Println("Let's go")
			parser.Next()
		case "openbracket":
			parser.Next()
		case "closebracket":
			break ARGSPAR
		default:
			value := parser.ParseValue()
			if len(value) == 0 && len(value) > 1 {
				throw("Argument has more than one value or is empty.", token.Position, token.Line)
			}

			args = append(args, value[0])
		}
	}

	return args
}

func (parser *Parser) ParseFuncCall() *FuncCall {
	funcCall := &FuncCall{}

FUNCPAR:
	for parser.CurrentPosition >= 0 {
		token := parser.CurrentToken

		switch token.Type {
		case "ident":
			funcCall.Func = newDataTypeNode(token)
			parser.Next("openbracket")
		case "openbracket":
			fmt.Println("Parsing args")
			funcCall.Arguments = parser.ParseCallArgs()
			fmt.Println("Parsed args")
		case "closebracket":
			parser.Next()
			break FUNCPAR
		}
	}

	return funcCall
}

func (parser *Parser) ParseBrackets() *Brackets {
	brackDec := &Brackets{}
	value := []Node{}

BRACKETPAR:
	for parser.CurrentPosition >= 0 {
		token := parser.CurrentToken

		switch token.Type {
		case "openbracket":
			parser.Next()
		case "closebracket":
			parser.Next()
			break BRACKETPAR
		default:
			value = parser.Parse(value)
		}
	}
	brackDec.Value = value

	return brackDec
}

func getTokenTypesExpects(token Token) []string {
	for k, expects := range tokenTypesExpects {
		types := strings.Split(k, ",")
		if !slices.Contains(types, token.Type) {
			continue
		}
		return expects
	}
	return []string{}
}

func (parser *Parser) ParseValue() []Node {
	value := []Node{}
	fmt.Println("\n\nParsing value...")
	nextExpects := []string{}
	for parser.CurrentPosition >= 0 {
		fmt.Println("Hello", parser.CurrentToken, nextExpects)
		currentToken := parser.CurrentToken
		if len(nextExpects) > 0 && !slices.Contains(nextExpects, currentToken.Type) {
			fmt.Println("\n\nParsing value ended\n\n")
			break
		}

		expects := getTokenTypesExpects(currentToken)
		if len(expects) > 0 {
			nextExpects = expects
		}

		value = parser.Parse(value)
	}

	return value
}

func (parser *Parser) AST() []Node {
	parser.CurrentPosition = -1
	parser.Expected = []string{}

	nodes := []Node{}

	parser.Next()
	for parser.CurrentPosition >= 0 {
		nodes = parser.Parse(nodes)
	}

	return nodes
}
