package main

type Node interface {
	Position() int
	Line() int
}

type Brackets struct {
	Value []Node
	X, Y  int
}

func (brackDec *Brackets) Position() int {
	return brackDec.X
}
func (brackDec *Brackets) Line() int {
	return brackDec.Y
}

type IdentNode struct {
	Value string
	X, Y  int
}

func (identNode *IdentNode) Position() int {
	return identNode.X
}
func (identNode *IdentNode) Line() int {
	return identNode.Y
}

type VarDec struct {
	Value      []Node
	Identifier IdentNode
	X, Y       int
}

func (varDec *VarDec) Position() int {
	return varDec.X
}
func (varDec *VarDec) Line() int {
	return varDec.Y
}

type SetVar struct {
	Var   *IdentNode
	Value []Node
	X, Y  int
}

func (setVar *SetVar) Position() int {
	return setVar.X
}
func (setVar *SetVar) Line() int {
	return setVar.Y
}

type SetElem struct {
	Elem  *GetElementNode
	Value []Node
	X, Y  int
}

func (setElem *SetElem) Position() int {
	return setElem.X
}
func (setElem *SetElem) Line() int {
	return setElem.Y
}

type FuncDec struct {
	Identifier IdentNode
	Arguments  []IdentNode
	Body       []Node
	Template   func(v ...any) any
	X, Y       int
}

func newFTemp(identifier string, t func(v ...any) any) *FuncDec {
	return &FuncDec{
		Identifier: IdentNode{Value: identifier},
		Template:   t,
	}
}

func (funcDec *FuncDec) Position() int {
	return funcDec.X
}
func (funcDec *FuncDec) Line() int {
	return funcDec.Y
}

type FuncCall struct {
	Func      Node
	Arguments []Node
	X, Y      int
}

type Argument struct {
	Ident IdentNode
	Value []Node
}

func (funcCall *FuncCall) Position() int {
	return funcCall.X
}
func (funcCall *FuncCall) Line() int {
	return funcCall.Y
}

type NumNode struct {
	Value float64
	X, Y  int
}

func (numNode *NumNode) Position() int {
	return numNode.X
}
func (numNode *NumNode) Line() int {
	return numNode.Y
}

type StrNode struct {
	Value string
	X, Y  int
}

func (strNode *StrNode) Position() int {
	return strNode.X
}
func (strNode *StrNode) Line() int {
	return strNode.Y
}

type BoolNode struct {
	Value bool
	X, Y  int
}

func (boolNode *BoolNode) Position() int {
	return boolNode.X
}
func (boolNode *BoolNode) Line() int {
	return boolNode.Y
}

type Element struct {
	Key   []Node
	Value []Node
	X, Y  int
}

func (elem *Element) Position() int {
	return elem.X
}
func (elem *Element) Line() int {
	return elem.Y
}

type MapNode struct {
	Map  []*Element
	X, Y int
}

func (mapNode *MapNode) Position() int {
	return mapNode.X
}
func (mapNode *MapNode) Line() int {
	return mapNode.Y
}

type GetElementNode struct {
	Map, Key []Node
	X, Y     int
}

func (getElementNode *GetElementNode) Position() int {
	return getElementNode.X
}
func (getElementNode *GetElementNode) Line() int {
	return getElementNode.Y
}

type IfStmt struct {
	Condition, Body []Node
	X, Y            int
}

func (ifStmt *IfStmt) Position() int {
	return ifStmt.X
}
func (ifStmt *IfStmt) Line() int {
	return ifStmt.Y
}

type BinOpNode struct {
	operator string
	L, R     Node
	X, Y     int
}

func (binOpNode *BinOpNode) Position() int {
	return binOpNode.X
}
func (binOpNode *BinOpNode) Line() int {
	return binOpNode.Y
}
