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
	Value      [][]Node
	Identifier []IdentNode
	Argument   bool //Interpreter only
	X, Y       int
}

func (varDec *VarDec) Position() int {
	return varDec.X
}
func (varDec *VarDec) Line() int {
	return varDec.Y
}

type NilNode struct {
	Value      []Node
	Identifier IdentNode
	X, Y       int
}

func (nilNode *NilNode) Position() int {
	return nilNode.X
}
func (nilNode *NilNode) Line() int {
	return nilNode.Y
}

type SetVar struct {
	Var   []IdentNode
	Value [][]Node
	X, Y  int
}

func (setVar *SetVar) Position() int {
	return setVar.X
}
func (setVar *SetVar) Line() int {
	return setVar.Y
}

type MultIdents struct {
	Idents []IdentNode
	X, Y   int
}

func (multIdents *MultIdents) Position() int {
	return multIdents.X
}
func (multIdents *MultIdents) Line() int {
	return multIdents.Y
}

type FuncDec struct {
	Identifier IdentNode
	Self       *StructObject //For interpreter
	Arguments  []IdentNode
	Body       []Node
	Template   func(v ...any) []any
	X, Y       int
}

func newFTemp(identifier string, t func(v ...any) []any) *FuncDec {
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

/*type NumNode struct {
	Value float64
	Int   bool
	X, Y  int
}

func (numNode *NumNode) Position() int {
	return numNode.X
}
func (numNode *NumNode) Line() int {
	return numNode.Y
}*/

type IntNode struct {
	Value int64
	X, Y  int
}

func (intNode *IntNode) Position() int {
	return intNode.X
}
func (intNode *IntNode) Line() int {
	return intNode.Y
}

type FloatNode struct {
	Value float64
	X, Y  int
}

func (floatNode *FloatNode) Position() int {
	return floatNode.X
}
func (floatNode *FloatNode) Line() int {
	return floatNode.Y
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

	X, Y int
}

type SetElem struct {
	Elem  *GetElementNode
	Value []Node

	X, Y int
}

func (setElem *SetElem) Position() int {
	return setElem.X
}
func (setElem *SetElem) Line() int {
	return setElem.Y
}

func (getElementNode *GetElementNode) Position() int {
	return getElementNode.X
}
func (getElementNode *GetElementNode) Line() int {
	return getElementNode.Y
}

type IfStmt struct {
	Condition, Body []Node
	Else            *ElseStmt
	X, Y            int
}

func (ifStmt *IfStmt) Position() int {
	return ifStmt.X
}
func (ifStmt *IfStmt) Line() int {
	return ifStmt.Y
}

type ElseStmt struct {
	Condition, Body []Node
	Else            *ElseStmt
	X, Y            int
}

func (elsestmt *ElseStmt) Position() int {
	return elsestmt.X
}
func (elsestmt *ElseStmt) Line() int {
	return elsestmt.Y
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

type WhileNode struct {
	Condition, Body []Node
	X, Y            int
}

func (wlNode *WhileNode) Position() int {
	return wlNode.X
}
func (wlNode *WhileNode) Line() int {
	return wlNode.Y
}

type ForeachNode struct {
	KeyIdent, ValueIdent IdentNode
	CycleValue           []Node
	Body                 []Node
	X, Y                 int
}

func (foreachNode *ForeachNode) Position() int {
	return foreachNode.X
}
func (foreachNode *ForeachNode) Line() int {
	return foreachNode.Y
}

type BreakNode struct {
	X, Y int
}

func (brNode *BreakNode) Position() int {
	return brNode.X
}
func (brNode *BreakNode) Line() int {
	return brNode.Y
}

type ContinueNode struct {
	X, Y int
}

func (cnNode *ContinueNode) Position() int {
	return cnNode.X
}
func (cnNode *ContinueNode) Line() int {
	return cnNode.Y
}

type ReturnNode struct {
	Value [][]Node
	X, Y  int
}

func (rtNode *ReturnNode) Position() int {
	return rtNode.X
}
func (rtNode *ReturnNode) Line() int {
	return rtNode.Y
}

type Import struct {
	Path []Node
	X, Y int
}

func (importNode *Import) Position() int {
	return importNode.X
}
func (importNode *Import) Line() int {
	return importNode.Y
}

type FieldDeclNode struct {
	Identifier IdentNode
	Bits       *IntNode
	Func       *FuncDec
}

type StructDeclNode struct {
	Identifier IdentNode
	X, Y       int

	Fields []*FieldDeclNode
}

func (structDecl *StructDeclNode) Position() int {
	return structDecl.X
}
func (structDecl *StructDeclNode) Line() int {
	return structDecl.Y
}

type FieldNode struct {
	Identifier IdentNode
	Value      []Node
}

type StructNode struct {
	Identifier IdentNode
	X, Y       int

	Fields []*FieldNode
}

func (structure *StructNode) Position() int {
	return structure.X
}
func (structure *StructNode) Line() int {
	return structure.Y
}

type GetFieldNode struct {
	Struct Node
	Field  []Node

	X, Y int
}

func (getField *GetFieldNode) Position() int {
	return getField.X
}
func (getField *GetFieldNode) Line() int {
	return getField.Y
}

type SetFieldNode struct {
	Field *GetFieldNode
	Value []Node

	X, Y int
}

func (setField *SetFieldNode) Position() int {
	return setField.X
}
func (setField *SetFieldNode) Line() int {
	return setField.Y
}

type GetPtrNode struct {
	Src Node

	X, Y int
}

func (getptr *GetPtrNode) Position() int {
	return getptr.X
}
func (getptr *GetPtrNode) Line() int {
	return getptr.Y
}
