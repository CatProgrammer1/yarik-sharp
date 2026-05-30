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
	"log"
	"runtime"
	"strconv"
	"syscall"
	"unsafe"
)

func floatIsInt(f float64) bool {
	return math.Trunc(f) == f
}

func twoDigitStr(str string) int {
	if len(str) < 2 {
		return int(str[0] - '0')
	}

	return int(str[0]-'0')*10 + int(str[1]-'0')
}

func argsCheck(v []any, min, max int, expectedDataTypes ...string) {
	if min == 0 && max == 0 {
		return
	}

	x, y := v[0].(int), v[1].(int)
	inter := v[2].(*Interpreter)

	if len(v) < min+BUILTIN_SPECIALS {
		throw(inter.CurrentFileName, "Attempt to pass less arguments to a function call than function actually need, minimum is %d.", x, y, min)
	} else if len(v) > max+3 {
		throw(inter.CurrentFileName, "Attempt to pass more arguments to a function call than function actually need, maximum is %d.", x, y, max)
	} else {
		args := v[BUILTIN_SPECIALS:]

		for i := 0; i < min; i++ {
			expectedDataType := expectedDataTypes[i]

			argument := args[i]

			if !checkDataType(expectedDataType, argument) {
				throw(inter.CurrentFileName, "Invalid argument #%d. Expected %s.", x, y, i+1, expectedDataType)
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
	case float32:
		return float64(n)
	case int64, int32, int, int16, int8:
		return float64(toInt64(n))
	case uint8, uint16, uint, uint32, uint64:
		return float64(toUint64(n))
	}
	return 0
}

func getValueType(v any) string {
	switch v := v.(type) {
	case nil:
		return "void"
	case string:
		return "string"
	case float32:
		return "f32"
	case float64:
		return "f64"
	case int64:
		return "i64"
	case int32:
		return "i32"
	case int16:
		return "i16"
	case int8:
		return "i8"
	case uint64:
		return "u64"
	case uint32:
		return "u32"
	case uint16:
		return "u16"
	case uint8:
		return "u8"
	case bool:
		return "bool"
	case *Map:
		return "table"
	case *StructObject:
		return instanceTypePrefix + v.Identifier
	case *Structure:
		return "structure"
	case *FuncDec:
		return "func"
	case uintptr, unsafe.Pointer:
		return "pointer"
	case error:
		return "error"
	}
	fmt.Printf("%T\n", v)
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
		case int64, float64, float32, int32, int16, int8, uint8, uint16, uint32, uint64:
			return true
		}

		return false
	case "usint":
		switch v.(type) {
		case int64, int32, int16, int8, uint8, uint16, uint32, uint64:
			return true
		}

		return false
	case "int":
		switch v.(type) {
		case int64, int32, int16, int8, uint8, uint16, uint32, uint64:
			return true
		}

		return false
	case "uint":
		switch v.(type) {
		case uint8, uint16, uint32, uint, uint64:
			return true
		}

		return false
	case "float":
		_, ok := v.(float64)
		if ok {
			return ok
		}

		_, ok = v.(float32)
		if ok {
			return ok
		}

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

func a() (int, int) {
	return 1, 1
}

func handle(err error) {
	if err != nil {
		panic(err)
	}
}

func handleLite(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func getFileString(path string) (string, error) {
	c, err := os.ReadFile(path)

	return string(c), err
}

func throw(filename, errForm string, x, y int, v ...any) {

	errMsg := fmt.Sprintf(shortennedPLName+" "+fmt.Sprintf("%s:%d:%d", filename, y, x)+": "+errForm, v...)
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

func loadLibraryIntoScope(interpreter_filename string, importPath string, node *ExternalImport, scope *Scope) { //go run yks run test.yks
	library := syscall.NewLazyDLL(importPath)
	err := library.Load()
	if err != nil {
		throw(interpreter_filename, err.Error(), node.X, node.Y)
	}

	suc := scope.Add(library, "DLL_LIBRARY", "string", node.X, node.Y)
	if !suc {
		throwNoPos("unsuccessfull")
	}

	name, _ := strings.CutSuffix(filepath.Base(library.Name), ".dll")
	scope.Data[name] = CLPTR(scope, &FuncDec{
		Identifier: IdentNode{
			Value: name,
			X:     node.X,
			Y:     node.Y,
		},
		Template: func(v ...any) []any {
			argsCheck(v, 1, 1, "string")

			x, y := v[0].(int), v[1].(int)

			v = v[BUILTIN_SPECIALS:]

			proc := library.NewProc(v[0].(string))
			err := proc.Find()
			if err != nil {
				throw(interpreter_filename, err.Error(), x, y)
			}

			suc := scope.Add(proc, "DLL_PROC", "string", x, y)
			if !suc {
				throwNoPos("unsuccessfull")
			}

			return []any{proc.Addr()}
		},
	}, node.X, node.Y)
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

	lexer := NewLexer(fileRel, content)
	tokens := lexer.GetTokens()

	if info {
		outputTokens(tokens)
	}

	parser := NewParser(fileRel, tokens)
	ast := parser.AST()

	interpreter := NewInterpreter(fileRel, ast)
	return interpreter.Complete(info)
}

func outputTokens(tokens []Token) {
	for k, token := range tokens {
		fmt.Printf("%d)'%s' - %s: %d,%d\n", k+1, fmt.Sprint(token.Value), token.Type, token.Line, token.Position)
	}
}

//!nasm -f bin s.asm -o test.bin

// ? go build -o bin/yks.exe yks
// *go run -race yks runinfo test.yks
func main() { //*go run yks run test.yks
	commands["build"] = func(args []string) {
		fmt.Println("Jly")
	}
	commands["run"] = func(args []string) {
		if len(args) == 0 {
			help([]string{})
			return
		}

		path := args[0]

		run(getAbsPath(path), path, false)
	}
	commands["runinfo"] = func(args []string) {
		if len(args) == 0 {
			help([]string{})
			return
		}

		path := args[0]

		run(getAbsPath(path), path, true)
	}
	commands["tokens"] = func(args []string) {
		if len(args) == 0 {
			help([]string{})
			return
		}

		path := args[0]

		src, err := getFileString(path)
		handleLite(err)

		lexer := NewLexer(path, src)

		outputTokens(lexer.GetTokens())
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
	args = args[1:]

	cmdFunc(args)
}
