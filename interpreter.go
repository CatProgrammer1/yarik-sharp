package main

import (
	"fmt"
	"math"
	"reflect"
	"regexp"
	"slices"
)

type Scope struct {
	Data   map[any]any
	Parent *Scope
}

func checkType[T any](v any) bool {
	_, ok := v.(T)
	return ok
}

var (
	builtinFuncs = map[string]func(v ...any) any{
		"print": func(v ...any) any {
			fmt.Println(v...)
			return nil
		},
	}
	typeName      = regexp.MustCompile(`<\*(?:[^.]+\.)?([^ ]+)`)
	binOperations = map[string]func(a, b any) any{
		"add": func(a, b any) any {
			return a.(float64) + b.(float64)
		},
		"sub": func(a, b any) any {
			return a.(float64) - b.(float64)
		},
		"div": func(a, b any) any {
			return a.(float64) / b.(float64)
		},
		"mul": func(a, b any) any {
			return a.(float64) * b.(float64)
		},
		"pow": func(a, b any) any {
			return math.Pow(a.(float64), b.(float64))
		},

		"greater": func(a, b any) any {
			return a.(float64) > b.(float64)
		},
		"less": func(a, b any) any {
			return a.(float64) < b.(float64)
		},
		"greatereq": func(a, b any) any {
			return a.(float64) >= b.(float64)
		},
		"lesseq": func(a, b any) any {
			return a.(float64) <= b.(float64)
		},

		"equals": func(a, b any) any {
			return a == b
		},
		"notequals": func(a, b any) any {
			return a != b
		},
	}
)

func getInterfaceType(v any) string {
	typeSM := typeName.FindAllStringSubmatch(reflect.ValueOf(v).String(), -1)
	if len(typeSM) == 0 {
		return ""
	}

	return typeSM[0][1]
}

func NewScope(parent *Scope) *Scope {
	return &Scope{
		Data:   make(map[any]any),
		Parent: parent,
	}
}

func (scope *Scope) Add(key, value any) (success bool) {
	if _, ok := scope.Data[key]; ok {
		return false
	}
	scope.Data[key] = value
	return true
}

func (scope *Scope) Set(key, value any) (success bool) {
	if _, ok := scope.Data[key]; ok {
		scope.Data[key] = value
		return true
	} else if scope.Parent != nil {
		return scope.Parent.Set(key, value)
	}
	return false
}

func (scope *Scope) Get(key any) any {
	v, ok := scope.Data[key]
	if ok {
		return v
	} else if scope.Parent != nil {
		return scope.Parent.Get(key)
	}
	return nil
}

type Interpreter struct {
	AST          []Node
	CurrentScope *Scope
}

func NewInterpreter(ast []Node) *Interpreter {
	return &Interpreter{
		AST: ast,
	}
}

func (inter *Interpreter) GetBinOpValue(node *BinOpNode) any {
	l, r := inter.GetNodeValue(node.L), inter.GetNodeValue(node.R)

	f := binOperations[node.operator]

	return f(l, r)
}

func (inter *Interpreter) GetTableValueByKeys(table any, keys []any, getElemN *GetElementNode, index int) any {
	if index >= len(keys) {
		return nil
	}

	key := keys[index]

	switch table := table.(type) {
	case map[any]any:
		val := table[key]

		if index+1 < len(keys) {
			return inter.GetTableValueByKeys(val, keys, getElemN, index+1)
		}
		return val
	}
	throw("Attempt to index non-table value", getElemN.X, getElemN.Y)
	return nil
}

func (inter *Interpreter) GetNodeValue(node Node) any {
	switch node := node.(type) {
	case *NumNode:
		return node.Value
	case *StrNode:
		return node.Value
	case *BoolNode:
		return node.Value
	case *BinOpNode:
		return inter.GetBinOpValue(node)
	case *Brackets:
		return inter.GetNodeValueS(node.Value)
	case *MapNode:
		fmt.Println("MAP")
		return inter.GetMap(node)
	case *FuncDec:
		return node
	case *FuncCall:
		return inter.CallFunction(node)
	case *IdentNode:
		return inter.CurrentScope.Get(node.Value)
	case *GetElementNode:
		tableNode, keyNodes := inter.GetTableAndKeys(node, []Node{})
		if tableNode == nil {
			throw("Attempt to index nothing", node.X, node.Y)
		}

		table := inter.GetNodeValue(tableNode)
		keys := []any{}
		for _, keyNode := range keyNodes {
			keys = append(keys, inter.GetNodeValue(keyNode))
		}

		switch table := table.(type) {
		case map[any]any:
			return inter.GetTableValueByKeys(table, keys, node, 0)
		default:
			throw("Cannot index non-table value", node.X, node.Y)
		}
	}
	throw("Invalid node '%s'.", node.Position(), node.Line(), getInterfaceType(node))
	return nil
}

func (inter *Interpreter) GetNodeValueS(nodes []Node) any {
	if len(nodes) > 1 || len(nodes) == 0 {
		throw("Value has more than one value or is empty", 0, 0)
	}
	return inter.GetNodeValue(nodes[0])
}

func (inter *Interpreter) GetTableAndKeys(node *GetElementNode, keys []Node) (Node, []Node) {
	if len(node.Map) > 1 {
		throw("Cannot index more than 1 value at the same time", node.X, node.Y)
	}
	if len(node.Key) > 1 {
		throw("Key cannot have more than 1 value", node.X, node.Y)
	}
	keys = append(keys, node.Key[0])
	switch mapNode := node.Map[0].(type) {
	case *GetElementNode:
		return inter.GetTableAndKeys(mapNode, keys)
	default:
		slices.Reverse(keys)
		return mapNode, keys
	}
}

func (inter *Interpreter) GetMap(node *MapNode) map[any]any {
	m := make(map[any]any)

	for _, element := range node.Map {
		fmt.Println("ASDDDDDDDDDDDDDDD")
		key, value := inter.GetNodeValueS(element.Key), inter.GetNodeValueS(element.Value)

		m[key] = value
	}

	return m
}

func (inter *Interpreter) Current(scope *Scope) {
	inter.CurrentScope = scope
}

func (inter *Interpreter) CallFunction(node *FuncCall) any {
	funcDec, ok := inter.GetNodeValue(node.Func).(*FuncDec)
	if !ok {
		throw("Attempt to call a non-function object.", node.X, node.Y)
	}

	if funcDec.Template != nil {
		args := []any{}
		for _, argNode := range node.Arguments {
			arg := inter.GetNodeValue(argNode)

			args = append(args, arg)
		}

		return funcDec.Template(args...)
	}

	body := funcDec.Body
	extraBody := []Node{}

	for i, argNode := range node.Arguments {
		if i+1 > len(node.Arguments) {
			throw("Attempt to pass more arguments to a function call than function actually need.", node.X, node.Y)
		}

		extraBody = append(extraBody, &VarDec{
			Identifier: funcDec.Arguments[i],
			Value:      []Node{argNode},
		})
	}

	_, _, value := inter.CompeleteBody(slices.Concat(extraBody, body))

	return value
}

func (inter *Interpreter) CompeleteBody(body []Node) (end, skip bool, value any) {
	scope := NewScope(inter.CurrentScope)
	inter.Current(scope)

	for _, node := range body {
		end, skip, value := inter.CompleteNode(node)
		if value != nil || end || skip {
			return end, skip, value
		}
	}

	inter.Current(scope.Parent)
	return false, false, nil
}

func (inter *Interpreter) SetTableElementValue(table map[any]any, keys []any, value any, index int) {
	if index >= len(keys) {
		return
	}

	key := keys[index]

	elem := table[key]
	switch elem := elem.(type) {
	case map[any]any:
		if index+1 >= len(keys) {
			table[key] = value
			break
		}
		inter.SetTableElementValue(elem, keys, value, index+1)
	default:
		table[key] = value
		fmt.Println(table)
	}
}

func (inter *Interpreter) SetElementValue(node *SetElem) {
	tableNode, keyNodes := inter.GetTableAndKeys(node.Elem, []Node{})
	if tableNode == nil {
		throw("Attempt to index nothing", node.X, node.Y)
	}

	table := inter.GetNodeValue(tableNode)
	keys := []any{}
	for _, keyNode := range keyNodes {
		keys = append(keys, inter.GetNodeValue(keyNode))
	}

	value := inter.GetNodeValueS(node.Value)

	switch table := table.(type) {
	case map[any]any:
		inter.SetTableElementValue(table, keys, value, 0)
	default:
		throw("Cannot index non-table value", node.X, node.Y)
	}
}

func (inter *Interpreter) CompleteNode(node Node) (end, skip bool, value any) {
	switch node := node.(type) {
	case *FuncDec:
		if len(node.Identifier.Value) == 0 {
			throw("Name of the function cannot be empty.", node.X, node.Y)
		}
		if !inter.CurrentScope.Add(node.Identifier.Value, node) {
			throw("Attempt to redeclare variable '%s'.", node.X, node.Y, node.Identifier.Value)
		}
	case *VarDec:
		fmt.Println(node.Identifier.Value)
		if !inter.CurrentScope.Add(node.Identifier.Value, inter.GetNodeValueS(node.Value)) {
			throw("Attempt to redeclare variable '%s'.", node.X, node.Y, node.Identifier.Value)
		}
	case *SetVar:
		if !inter.CurrentScope.Set(node.Var.Value, inter.GetNodeValueS(node.Value)) {
			throw("Attempt to change value of non-existing variable '%s'.", node.X, node.Y, node.Var.Value)
		}
	case *FuncCall:
		inter.GetNodeValue(node)	
	case *SetElem:
		inter.SetElementValue(node)
	default:
		throw("Invalid node '%s'.", node.Position(), node.Line(), getInterfaceType(node))
	}
	return false, false, nil
}

func (inter *Interpreter) Complete(info bool) {
	mainScope := NewScope(nil)

	inter.CurrentScope = mainScope

	for ident, function := range builtinFuncs {
		mainScope.Add(ident, newFTemp(ident, function))
	}

	for _, node := range inter.AST {
		inter.CompleteNode(node)
	}

	if info {
		fmt.Println(mainScope.Data)
	}
}
