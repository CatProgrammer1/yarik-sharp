package main

import (
	"C"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/elliotchance/orderedmap/v3"
)

const (
	plName           = "Yarik#"
	shortennedPLName = "yks"

	fileType            = ".yks"
	major, minor, patch = 1, 6, 2
	stage               = "beta"
)

var (
	args     = os.Args[1:]
	commands = make(map[string]func(args []string))
	libs     = filepath.Join(getParentPath(getParentPath(getSelfPath())), "src")
)

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

func floatToPtr(f float64) uintptr {
	return uintptr(math.Float64bits(f))
}

func ptrToFloat(p uintptr) float64 {
	float := math.Float64frombits(uint64(p))
	if math.IsNaN(float) {
		return -1
	}

	return float
}

func numberToFloat64(n any) (float64, bool) {
	v := reflect.ValueOf(n)
	switch v.Kind() {
	case reflect.Float32, reflect.Float64:
		return v.Convert(reflect.TypeOf(float64(0))).Float(), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(v.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(v.Uint()), true
	default:
		return -1, false
	}
}

func mustNTOF64(n any) float64 {
	v := reflect.ValueOf(n)
	switch v.Kind() {
	case reflect.Float32, reflect.Float64:
		return v.Convert(reflect.TypeOf(float64(0))).Float()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(v.Uint())
	default:
		return 0
	}
}

func getValueType(v any) string {
	switch v.(type) {
	case nil:
		return "void"
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "bool"
	case *orderedmap.OrderedMap[any, any]:
		return "table"
	case *StructObject:
		return "instance"
	case *Structure:
		return "structure"
	case *FuncDec:
		return "func"
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
		_, ok := v.(PTR)

		return ok
	case "number":
		oks := []bool{}

		_, ok := v.(float64)
		oks = append(oks, ok)

		_, ok = v.(uintptr)
		oks = append(oks, ok)

		_, ok = v.(PTR)
		oks = append(oks, ok)

		for _, ok := range oks {
			if ok {
				return ok
			}
		}
		return false
	case "bool":
		_, ok := v.(bool)

		return ok
	case "table":
		_, ok := v.(*orderedmap.OrderedMap[any, any])

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
func main() { //*go run yks runinfo test.yks
	/*dll := syscall.NewLazyDLL("test_struct.dll")
	proc := dll.NewProc("check_OBJECT_ATTRIBUTES")

	utf16S, _ := syscall.UTF16FromString("Sigma")

	unicode_string := newInstance("UNICODE_STRING",
		[]*Field{
			{"Length", false, -16, float64(len(utf16S) * 2)},
			{"MaximumLength", false, 16, float64(cap(utf16S) * 2)},
			{"Buffer", false, 0, PTR(ptrToFloat(uintptr(unsafe.Pointer(&utf16S[0]))))},
		})

	fmt.Println(float64(int32(-1)))

	obj := newInstance("OBJECT_ATTRIBUTES",
		[]*Field{
			{"Length", false, 64, float64(48)},
			{"RootDirectory", false, 0, float64(0)},
			{"ObjectName", false, 0, unicode_string},
			{"Attributes", false, 64, float64(64)},
			{"SecurityDescriptor", false, 0, float64(0)},
			{"SecurityQoS", false, 0, float64(0)},
		})

	fmt.Println(obj.Fields[0].Value)

	layout := obj.Layout()

	proc.Call(uintptr(unsafe.Pointer(&obj.ToMemoryLayout(layout)[0])))

	obj.FromMemoryLayout(layout)
	fmt.Println(obj.Fields[0].Value)*/

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
