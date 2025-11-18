package main

import (
	"slices"
	"strings"
)

const (
	tableSeparatorTokenType      = "comma"
	tableKeyValueAssignTokenType = "colon"
)

var (
	tokenTypesExpects = map[string][]string{
		"number,ident,string,bool,nil,openbracket,opensqbrac,newstruct": {"openbracket", "opensqbrac", "add", "bitor", "sub", "div", "mul", "pow",
			"equals", "notequals", "greater", "less", "greatereq", "lesseq", "and", "or", "indexstruct"},
		"add,sub,div,mul,pow,equals,notequals,greater,less,greatereq,lesseq,bitor,and,or,return,getptr": {"number", "ident", "string", "bool", "openbracket", "nil"},
		"opensqbrac": {"opensqbrac"},
	}
	binOpsList = []string{
		"add", "sub", "div", "mul", "pow",
		"equals", "notequals", "greater", "less", "greatereq", "lesseq",
		"bitor",
		"and", "or",
	}
)

type Parser struct {
	CurrentToken    Token
	LastToken       Token
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

func tokenIsBinOp(token Token) bool {
	return slices.Contains(binOpsList, token.Type)
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
		throw("Expected '%s' got '%s'.", currentToken.Position, currentToken.Line, parser.Expected, currentToken.Type)
	}
	if len(parser.Unexpected) > 0 && slices.Contains(parser.Unexpected, currentToken.Type) {
		throw("Invalid token '%s'.", currentToken.Position, currentToken.Line, parser.Expected, currentToken.Type)
	}

	parser.Unexpected = []string{}
	parser.CurrentToken = currentToken

	if parser.LastToken == parser.CurrentToken {
		throw("Invalid token '%s'.", currentToken.Position, currentToken.Line, parser.Expected, currentToken.Type)
	}
	parser.LastToken = parser.CurrentToken
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
	//case *IdentNode, *NumNode, *BoolNode, *StrNode, *FuncCall:
	default:
		return rOp
	}
}

func setLastRightOperand(binOpNode *BinOpNode, value Node) {
	switch rOp := binOpNode.R.(type) {
	case *BinOpNode:
		setLastRightOperand(rOp, value)
	//case *IdentNode, *NumNode, *BoolNode, *StrNode, nil:
	default:
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
	case "nil":
		return &NilNode{}
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
	case *GetPtrNode:
		if lastNode.Src == nil {
			lastNode.Src = node
			return nodes
		}
		nodes = append(nodes, node)
	default:
		nodes = append(nodes, node)
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

func (parser *Parser) Parse(nodes []Node, bodyParsing bool) []Node {
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
	case "wlloop":
		wlLoop := parser.ParseWhileLoop()
		wlLoop.X, wlLoop.Y = x, y

		nodes = append(nodes, wlLoop)
		return nodes
	case "forloop":
		foreachLoop := parser.ParseForeachLoop()
		foreachLoop.X, foreachLoop.Y = x, y

		nodes = append(nodes, foreachLoop)
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
	case "break":
		nodes = append(nodes, &BreakNode{x, y})
		parser.Next()
		return nodes
	case "continue":
		nodes = append(nodes, &ContinueNode{x, y})
		parser.Next()
		return nodes
	case "return":
		parser.Next()
		nodes = append(nodes, &ReturnNode{
			parser.ParseReturnValue(),
			x, y,
		})

		return nodes
	case "import":
		parser.Next()
		nodes = append(nodes, &Import{
			parser.ParseValue(),
			x, y,
		})

		return nodes
	case "indexstruct":
		lastNode := getLastNode(nodes)
		if lastNode != nil {
			switch lastNode := lastNode.(type) {
			case *GetFieldNode, *GetElementNode, *IdentNode:
				parser.Next("ident")

				node := &GetFieldNode{
					lastNode,
					parser.Parse([]Node{}, false),

					x, y,
				}

				return replaceLastNodeWith(nodes, node)
			case *BinOpNode:
				parser.Next("ident")

				rightOpNode := getLastRightOperand(lastNode)
				if rightOpNode != nil {
					node := &GetFieldNode{
						rightOpNode,
						parser.ParseValue(),
						x, y,
					}

					setLastRightOperand(lastNode, node)
					return nodes
				}
			case *GetPtrNode:
				parser.Next("ident")

				src := lastNode.Src
				if src != nil {
					node := &GetFieldNode{
						src,
						parser.ParseValue(),
						x, y,
					}
					lastNode.Src = node

					
					return nodes
				}
			}
		}
	case "newstruct":
		nodes = append(nodes, parser.ParseNewStruct())

		return nodes
	case "struct":
		nodes = append(nodes, parser.ParseStructDecl())

		return nodes
	case "cmtopen":
		parser.SkipComment()

		return nodes
	case "assign":
		lastNode := getLastNode(nodes)
		if lastNode != nil {
			switch lastNode := lastNode.(type) {
			case *IdentNode:
				parser.Next()

				node := &SetVar{
					[]IdentNode{*lastNode},
					[][]Node{parser.ParseValue()},
					x, y,
				}

				return replaceLastNodeWith(nodes, node)
			case *MultIdents:

				node := &SetVar{
					lastNode.Idents,
					parser.ParseMultValues(),
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
			case *GetFieldNode:
				parser.Next()

				node := &SetFieldNode{
					lastNode,
					parser.ParseValue(),

					x, y,
				}

				return replaceLastNodeWith(nodes, node)
			}
		}
	case "add", "sub", "div", "mul", "pow",
		"equals", "notequals", "greater", "less", "greatereq", "lesseq",
		"bitor",
		"and", "or":
		lastNode := getLastNode(nodes)
		if lastNode == nil {
			if currentToken.Type == "sub" {
				parser.Next()
				return append(nodes, &BinOpNode{
					operator: currentToken.Type,
					X:        x,
					Y:        y,
				})
			}
			throw("Expected left operand for '%s' binary operation got nothing nigger.", x, y, currentToken.Type)
		} else {

			binOpNode := &BinOpNode{
				operator: currentToken.Type,
				X:        x,
				Y:        y,
			}
			switch node := lastNode.(type) {
			case *BinOpNode:
				switch node.operator {
				case "mul", "div", "pow", "bitor":
					binOpNode.L = node
					nodes = replaceLastNodeWith(nodes, binOpNode)
				default:
					switch currentToken.Type {
					case "equals", "notequals", "greater", "less", "greatereq", "lesseq":
						binOpNode.L = node
						nodes = replaceLastNodeWith(nodes, binOpNode)
						parser.Next()
						return nodes
					}

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
	case "getptr":
		parser.Next()

		getPtrNode := &GetPtrNode{
			X: x, Y: y,

			Src: nil,
		}

		nodes = append(nodes, getPtrNode)
		return nodes
	case "openbracket":
		lastNode := getLastNode(nodes)

		switch lastNode := lastNode.(type) {
		case *BinOpNode:
			R := getLastRightOperand(lastNode)

			if R != nil {
				funcCall := parser.ParseFuncCall()
				funcCall.Func = R
				funcCall.X, funcCall.Y = x, y

				setLastRightOperand(lastNode, funcCall)
				return nodes
			}
		case *IdentNode, *GetFieldNode:
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
		case *GetPtrNode:
			src := lastNode.Src
			if src != nil {
				lastNode.Src = &GetElementNode{
					[]Node{lastNode.Src},
					parser.ParseKey(),
					x, y,
				}
				return nodes
			}
		case nil:
			nodes = appendDataType(parser.ParseMap(), nodes)

			return nodes
		default:
			nodes = replaceLastNodeWith(nodes, &GetElementNode{
				[]Node{lastNode},
				parser.ParseKey(),
				x, y,
			})

			return nodes
		}
	case "ident":
		nextToken := parser.PeekNext()

		switch nextToken.Type {
		case "comma":
			if bodyParsing {
				multAssign := parser.ParseMultipleIdents()

				nodes = append(nodes, multAssign)

				return nodes
			}
			fallthrough
		default:
			nodes = appendDataType(newDataTypeNode(currentToken), nodes)
			parser.Next()

			return nodes
		}
	case "number", "string", "bool", "nil":
		nodes = appendDataType(newDataTypeNode(currentToken), nodes)
		parser.Next()

		return nodes
	}

	throw("Invalid token '%s'.", currentToken.Position, currentToken.Line, currentToken.Type)
	return nil
}

func (parser *Parser) ParseMultipleIdents() *MultIdents {
	idents := []IdentNode{}
	x, y := parser.CurrentToken.Position, parser.CurrentToken.Line

MULT_ASSIGN_PAR:
	for parser.CurrentPosition >= 0 {
		token := parser.CurrentToken

		switch token.Type {
		case "ident":
			idents = append(idents, IdentNode{
				Value: token.Value.(string),
				X:     token.Position, Y: token.Line,
			})

			parser.Unexpect("ident")
			parser.Next()
		case "comma":
			parser.Unexpect("comma")
			parser.Next("ident")
		default:
			break MULT_ASSIGN_PAR
		}
	}

	return &MultIdents{
		Idents: idents,
		X:      x,
		Y:      y,
	}
}

func (parser *Parser) ParseMultValues() [][]Node {
	values := [][]Node{}

MULTVAL_PAR:
	for parser.CurrentPosition >= 0 {
		token := parser.CurrentToken

		switch token.Type {
		case "comma", "assign":
			parser.Unexpect("comma")
			parser.Next()

			values = append(values, parser.ParseValue())
		default:
			break MULTVAL_PAR
		}
	}

	return values
}

func (parser *Parser) ParseIfStmt() *IfStmt {
	ifStmt := &IfStmt{}
	condition := []Node{}

STMTPAR:
	for parser.CurrentPosition >= 0 {
		token := parser.CurrentToken

		switch token.Type {
		case "ifstmt":
			parser.Next()
		case "openbrace":
			ifStmt.Body = parser.ParseBody()

			token = parser.CurrentToken
			if token.Type != "else" {
				break STMTPAR
			}
			elseStmt := &ElseStmt{
				X: token.Position, Y: token.Line,
			}
			parser.Next("openbrace", "ifstmt")

			token = parser.CurrentToken
			switch token.Type {
			case "ifstmt":
				elseifStmt := parser.ParseIfStmt()

				elseStmt.Condition = elseifStmt.Condition
				elseStmt.Body = elseifStmt.Body
				elseStmt.Else = elseifStmt.Else
			case "openbrace":
				elseStmt.Body = parser.ParseBody()
			}

			ifStmt.Else = elseStmt
			break STMTPAR
		default:
			condition = parser.Parse(condition, false)
		}
	}
	ifStmt.Condition = condition

	return ifStmt
}

func (parser *Parser) ParseForeachLoop() *ForeachNode {
	foreachNode := &ForeachNode{}

FOREACHPAR:
	for parser.CurrentPosition >= 0 {
		token := parser.CurrentToken

		switch token.Type {
		case "forloop":
			parser.Next("ident")
		case "ident":
			if len(foreachNode.KeyIdent.Value) == 0 {
				foreachNode.KeyIdent = IdentNode{
					Value: token.Value.(string),
					X:     token.Position, Y: token.Line,
				}
				parser.Next("comma")
			} else {
				foreachNode.ValueIdent = IdentNode{
					Value: token.Value.(string),
					X:     token.Position, Y: token.Line,
				}
				parser.Next("assign")
			}
		case "comma":
			parser.Next("ident")
		case "openbrace":
			foreachNode.Body = parser.ParseBody()
			break FOREACHPAR
		case "assign":
			parser.Next()
			foreachNode.CycleValue = parser.ParseValue()

			token = parser.CurrentToken
			if token.Type == "openbrace" {
				break
			}
			fallthrough
		default:
			throw("Invalid token '%s'.", token.Position, token.Line, token.Type)
		}
	}

	return foreachNode
}

func (parser *Parser) ParseWhileLoop() *WhileNode {
	wlNode := &WhileNode{}
	condition := []Node{}

WHILEPAR:
	for parser.CurrentPosition >= 0 {
		token := parser.CurrentToken

		switch token.Type {
		case "wlloop":
			parser.Next()
		case "openbrace":
			wlNode.Body = parser.ParseBody()
			break WHILEPAR
		default:
			condition = parser.Parse(condition, false)
		}
	}
	wlNode.Condition = condition

	return wlNode
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
			key = parser.Parse(key, false)
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

	elementCount := float64(0)

MAPPAR:
	for parser.CurrentPosition >= 0 {
		token := parser.CurrentToken

		switch token.Type {
		case tableKeyValueAssignTokenType:
			parser.Unexpect(tableSeparatorTokenType, "closesqbrac")
			parser.Next()
			currentValue = parser.ParseValue()

			currenToken := parser.CurrentToken
			if currenToken.Type != tableSeparatorTokenType {
				throw("Expected comma after element value", currenToken.Position, currenToken.Line)
			}
		case tableSeparatorTokenType:
			if len(currentKey) > 1 || len(currentKey) == 0 {
				throw("Key has more than one value or empty.", token.Position, token.Line)
			}

			if len(currentValue) > 1 || len(currentValue) == 0 {
				throw("Element has more than one value or empty.", token.Position, token.Line)
			}
			mapNode.Map = append(mapNode.Map, &Element{
				Key:   currentKey,
				Value: currentValue,
			})
			currentKey = []Node{}
			parser.Unexpect(tableKeyValueAssignTokenType)
			parser.Next()

			elementCount++
		case "opensqbrac":
			parser.Unexpect(tableSeparatorTokenType, tableKeyValueAssignTokenType)
			parser.Next()
		case "closesqbrac":
			parser.Next()
			break MAPPAR
		default:
			currentKey = parser.ParseValue()
			token = parser.CurrentToken

			switch token.Type {
			case tableSeparatorTokenType:
				currentValue = currentKey
				currentKey = []Node{newDataTypeNode(Token{
					Value: elementCount,
					Type:  "number",
				})}
			}
		}
	}

	return mapNode
}

func (parser *Parser) ParseNewStruct() *StructNode {
	structure := &StructNode{
		X: parser.CurrentToken.Position,
		Y: parser.CurrentToken.Line,

		Fields: []*FieldNode{},
	}

STRUCTPAR:
	for parser.CurrentPosition >= 0 {
		token := parser.CurrentToken

		switch token.Type {
		case "newstruct":
			parser.Next("ident")
		case "ident":
			structure.Identifier = IdentNode{
				token.Value.(string),
				token.Position, token.Line,
			}
			parser.Next("openbrace")
		case "openbrace":
			parser.Next()
			structure.Fields = parser.ParseNewStructFields()
			break STRUCTPAR
		default:
			throw("Invalid token '%s'.", token.Position, token.Line, parser.Expected, token.Type)
		}
	}

	return structure
}

func (parser *Parser) ParseNewStructFields() []*FieldNode {
	fields := []*FieldNode{}

	var identifier IdentNode

FIELDS:
	for parser.CurrentPosition >= 0 {
		token := parser.CurrentToken

		switch token.Type {
		case "ident":
			identifier = IdentNode{
				token.Value.(string),
				token.Position, token.Line,
			}

			parser.Next(tableKeyValueAssignTokenType)
		case tableKeyValueAssignTokenType:
			parser.Unexpect(tableSeparatorTokenType, tableKeyValueAssignTokenType)
			parser.Next()

			fields = append(fields, &FieldNode{
				Identifier: identifier,
				Value:      parser.ParseValue(),
			})

			token = parser.CurrentToken
			if token.Type != tableSeparatorTokenType {
				throw("Expected '%s' after field value.", token.Position, token.Line, tableSeparatorTokenType)
			}

			identifier = IdentNode{}
		case tableSeparatorTokenType:
			parser.Next("ident", "closebrace")
		case "closebrace":
			parser.Next()
			break FIELDS
		default:
			throw("Invalid token '%s'.", token.Position, token.Line, parser.Expected, token.Type)
		}
	}

	return fields
}

func (parser *Parser) ParseStructDecl() *StructDeclNode {
	structDecl := &StructDeclNode{
		X: parser.CurrentToken.Position,
		Y: parser.CurrentToken.Line,

		Fields: []*FieldDeclNode{},
	}

STRUCTDECLPAR:
	for parser.CurrentPosition >= 0 {
		token := parser.CurrentToken

		switch token.Type {
		case "struct":
			parser.Next("ident")
		case "ident":
			structDecl.Identifier = IdentNode{
				token.Value.(string),
				token.Position, token.Line,
			}
			parser.Next("openbrace")
		case "openbrace":
			parser.Next()
			structDecl.Fields = parser.ParseStructDeclFields()
			break STRUCTDECLPAR
		default:
			throw("Invalid token '%s'.", token.Position, token.Line, parser.Expected, token.Type)
		}
	}

	return structDecl
}

func (parser *Parser) ParseStructDeclFields() []*FieldDeclNode {
	fields := []*FieldDeclNode{}

	var cbits *NumNode

FIELDS:
	for parser.CurrentPosition >= 0 {
		token := parser.CurrentToken

		switch token.Type {
		case "valbitcount":
			parser.Next("number")
			token = parser.CurrentToken

			cbits = newDataTypeNode(token).(*NumNode)
			parser.Next("ident")
		case "ident":
			fieldDeclNode := &FieldDeclNode{
				Identifier: IdentNode{
					token.Value.(string),
					token.Position, token.Line,
				},
				Func: nil,
			}
			if cbits != nil {
				fieldDeclNode.Bits = cbits
			}
			cbits = nil

			fields = append(fields, fieldDeclNode)

			parser.Next("comma")
		case "func":
			funcDecl := parser.ParseFuncDecl()

			fields = append(fields, &FieldDeclNode{
				Identifier: funcDecl.Identifier,
				Func:       funcDecl,
			})
		case "comma":
			parser.Next("ident", "func", "closebrace", "valbitcount")
		case "closebrace":
			parser.Next()
			break FIELDS
		default:
			throw("Invalid token '%s'.", token.Position, token.Line, parser.Expected, token.Type)
		}
	}

	return fields
}

func (parser *Parser) ParseBody() []Node {
	body := []Node{}

BODYPAR:
	for parser.CurrentPosition >= 0 {
		token := parser.CurrentToken

		switch token.Type {
		case "openbrace":
			parser.Next()
		case "closebrace":
			parser.Next()
			break BODYPAR
		default:
			body = parser.Parse(body, true)
		}
	}

	return body
}

func (parser *Parser) SkipComment() {

COMMENT:
	for parser.CurrentPosition >= 0 {
		token := parser.CurrentToken

		switch token.Type {
		case "cmtclose":
			parser.Next()
			break COMMENT
		default:
			parser.Next()
		}
	}
}

func (parser *Parser) ParseVariable() *VarDec {
	varDec := &VarDec{}

	values := false

VARPAR:
	for parser.CurrentPosition >= 0 {
		token := parser.CurrentToken
		x, y := token.Position, token.Line

		switch token.Type {
		case "var":
			parser.Next("ident")
		case "ident":
			varDec.Identifier = append(varDec.Identifier, IdentNode{token.Value.(string), x, y})
			parser.Next("assign", "comma")
		case "comma":
			parser.Next()
			if values {
				varDec.Value = append(varDec.Value, parser.ParseValue())
			}
		case "assign":
			values = true
			parser.Next()
			varDec.Value = append(varDec.Value, parser.ParseValue())

			token := parser.CurrentToken
			if token.Type != "comma" {
				break VARPAR
			}
		default:
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

		switch token.Type {
		case "comma":
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
			parser.Next()
			funcCall.Arguments = parser.ParseCallArgs()
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
			if tokenIsBinOp(token) {
				value = parser.Parse(value, false)
			} else {
				value = append(value, parser.ParseValue()...)
			}
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

func (parser *Parser) ParseReturnValue() [][]Node {
	values := [][]Node{}
	//value := []Node{}

	//nextExpects := getTokenTypesExpects(Token{Type: "return"})
PAR_RETURNVAL:
	for parser.CurrentPosition >= 0 {
		currentToken := parser.CurrentToken

		switch currentToken.Type {
		case "comma":
			parser.Unexpect("comma")
			parser.Next()
		default:
			/*if len(nextExpects) > 0 && !slices.Contains(nextExpects, currentToken.Type) {
				values = append(values, value)
				value = []Node{}
			}

			expects := getTokenTypesExpects(currentToken)
			if len(expects) > 0 {
				nextExpects = expects
			}

			value = parser.Parse(value)*/

			values = append(values, parser.ParseValue())

			currentToken = parser.CurrentToken
			if currentToken.Type != "comma" {
				break PAR_RETURNVAL
			}
		}
	}

	return values
}

func (parser *Parser) ParseValue() []Node {
	value := []Node{}
	nextExpects := []string{}
	for parser.CurrentPosition >= 0 {
		currentToken := parser.CurrentToken
		if len(nextExpects) > 0 && !slices.Contains(nextExpects, currentToken.Type) {
			break
		}

		expects := getTokenTypesExpects(currentToken)

		if len(expects) > 0 {
			nextExpects = expects
		}

		value = parser.Parse(value, false)
	}

	return value
}

func (parser *Parser) AST() []Node {
	parser.CurrentPosition = -1
	parser.Expected = []string{}

	nodes := []Node{}

	parser.Next()
	for parser.CurrentPosition >= 0 {
		nodes = parser.Parse(nodes, true)
	}

	return nodes
}
