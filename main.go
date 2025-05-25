package main

import (
	"fmt"
	"os"
)

func handle(err error) {
	if err != nil {
		panic(err)
	}
}

func getFileString(path string) string {
	c, err := os.ReadFile(path)
	handle(err)

	return string(c)
}

func throw(errForm string, x, y int, v ...any) {
	fmt.Printf("yks "+fmt.Sprintf("%d:%d", y, x)+": "+errForm+"\n", v...)
	os.Exit(1)
}

func main() { //go run main.go lexer.go parser.go nodes.go interpreter.go
	lexer := NewLexer(getFileString("test.yks"))
	fmt.Println("Tokenization...")
	tokens := lexer.GetTokens()
	fmt.Println("Tokenization ended:", tokens, "\n\n\n\n")

	fmt.Println("Parsing...")
	parser := NewParser(tokens)
	ast := parser.AST()
	fmt.Println("Parsing ended:", ast, "\n\n\n\n")
	//fmt.Println("AST:",ast[1].(*SetElem).Elem.Map[0].(*GetElementNode).Key[0])

	fmt.Println("Running...")
	interpreter := NewInterpreter(ast)
	interpreter.Complete(true)
}
