package main

import (
	"C"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
)
import (
	"runtime"
	"strconv"
	"unsafe"
)

const (
	plName           = "Yarik#"
	shortennedPLName = "yks"

	fileType            = ".yks"
	major, minor, patch = 0, 7, 3
	stage               = "beta"
)

var (
	args     = os.Args[1:]
	commands = make(map[string]func(args []string))
	libs     = filepath.Join(getParentPath(getParentPath(getSelfPath())), "src")
)

func floatIsInt(f float64) bool {
	return math.Trunc(f) == f
}

func argsCheck(v []any, min, max int, expectedDataTypes ...string) {
	if min == 0 && max == 0 {
		return
	}

	x, y := v[0].(int), v[1].(int)

	if len(v) < min+3 {
		throw("Attempt to pass less arguments to a function call than function actually need, minimum is %d.", x, y, min)
	} else if len(v) > max+3 {
		throw("Attempt to pass more arguments to a function call than function actually need, maximum is %d.", x, y, max)
	} else {
		args := v[3:]

		for i := 0; i < min; i++ {
			expectedDataType := expectedDataTypes[i]

			argument := args[i]

			if !checkDataType(expectedDataType, argument) {
				throw("Invalid argument #%d. Expected %s.", x, y, i+1, expectedDataType)
			}
		}
	}
}

func numtostr(v any) string {
	var s string

	switch v := v.(type) {
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', 32, 64)
	}

	return s
}

func numberToFloat64(n any) (float64, bool) {
	switch n := n.(type) {
	case float64:
		return n, true
	case int64:
		return float64(n), true
	}
	return 0, false
}

func numberToInt(n any) (int64, bool) {
	switch n := n.(type) {
	case float64:
		return int64(n), true
	case int64:
		return n, true
	}
	return 0, false
}

func mustNTOF64(n any) float64 {
	switch n := n.(type) {
	case float64:
		return n
	case int64:
		return float64(n)
	}
	return 0
}

func getValueType(v any) string {
	switch v.(type) {
	case nil:
		return "void"
	case string:
		return "string"
	case float64:
		return "float"
	case int64:
		return "int"
	case bool:
		return "bool"
	case *Map:
		return "table"
	case *StructObject:
		return "instance"
	case *Structure:
		return "structure"
	case *FuncDec:
		return "func"
	case uintptr, unsafe.Pointer:
		return "pointer"
	case error:
		return "error"
	}
	return "any"
}

func checkDataType(expected string, v any) bool {
	switch expected {
	case "any":
		return true
	case "string":
		_, ok := v.(string)

		return ok
	case "ptr":
		switch v.(type) {
		case uintptr, unsafe.Pointer:
			return true
		}

		return false
	case "number":
		switch v.(type) {
		case int64, float64:
			return true
		}

		return false
	case "int":
		_, ok := v.(int64)

		return ok
	case "float":
		_, ok := v.(float64)

		return ok
	case "bool":
		_, ok := v.(bool)

		return ok
	case "table":
		_, ok := v.(*Map)

		return ok
	case "instancestrict":
		_, ok := v.(*StructObject)

		return ok
	case "instance":
		if v == nil {
			return true
		}
		_, ok := v.(*StructObject)

		return ok
	case "structure":
		_, ok := v.(*Structure)

		return ok
	case "func":
		_, ok := v.(*FuncDec)

		return ok
	}
	return false
}

func getSelfPath() string {
	p, err := os.Executable()
	handle(err)

	return p
}

func help([]string) {
	fmt.Println("|Commands:")
	for k := range commands {
		fmt.Printf("|-- %s --\n", k)
	}
}

func handle(err error) {
	if err != nil {
		panic(err)
	}
}

func getFileString(path string) (string, error) {
	c, err := os.ReadFile(path)

	return string(c), err
}

func throw(errForm string, x, y int, v ...any) {

	errMsg := fmt.Sprintf(shortennedPLName+" "+fmt.Sprintf("%d:%d", y, x)+": "+errForm, v...)
	if !strings.HasSuffix(errMsg, ".") {
		errMsg += "."
	}
	if strings.Contains(errMsg, "non") && !strings.Contains(errMsg, "non-") {
		errMsg = strings.ReplaceAll(errMsg, "non", "non-")
	}

	fmt.Println(errMsg)
	os.Exit(1)
}

func throwNoPos(errForm string, v ...any) {
	fmt.Printf(shortennedPLName+": "+errForm+"\n", v...)
	os.Exit(1)
}

func getParentPath(path string) string {
	return filepath.Dir(path)
}

func getAbsPath(relPath string) string {
	abs, err := filepath.Abs(relPath)
	handle(err)

	return abs
}

func run(fileAbs, fileRel string, info bool) map[any]*Cell {
	if !strings.HasSuffix(fileAbs, fileType) {
		fileAbs += fileType
	}

	content, err := getFileString(fileAbs)
	if err != nil {
		throwNoPos(err.Error())
	}

	filesBeingUsed = append(filesBeingUsed, [2]string{fileAbs, fileRel})

	lexer := NewLexer(content)
	tokens := lexer.GetTokens()

	parser := NewParser(tokens)
	ast := parser.AST()

	interpreter := NewInterpreter(ast)
	return interpreter.Complete(info)
}

//!nasm -f bin s.asm -o test.bin

// ? go build -o bin/yks.exe yks
// *go run -race yks runinfo test.yks
func main() { //*go run yks runinfo test.yks
	commands["build"] = func(args []string) {
		//Lox
		fmt.Println("Kys")
	}
	commands["run"] = func(args []string) {
		path := args[0]

		run(getAbsPath(path), path, false)
	}
	commands["runinfo"] = func(args []string) {
		path := args[0]

		run(getAbsPath(path), path, true)
	}
	commands["tokens"] = func(args []string) {
		path := args[0]

		src, err := getFileString(path)
		handle(err)

		lexer := NewLexer(src)

		fmt.Println(lexer.GetTokens())
	}
	commands["version"] = func(args []string) {
		fmt.Printf("%s version %s%d.%d.%d-%s", plName, shortennedPLName, major, minor, patch, stage)
	}
	commands["arch"] = func(args []string) {
		fmt.Println(runtime.GOARCH)
	}
	commands["help"] = help

	if len(args) <= 0 {
		help([]string{})
		return
	}

	cmd := args[0]

	cmdFunc, ok := commands[cmd]
	if !ok {
		help([]string{})
		return
	}
	cmdFunc(args[1:])
}
