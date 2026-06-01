package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"unsafe"

	"github.com/elliotchance/orderedmap/v3"
)

var (
	externalCallingChan = make(chan ExternalTask, 20048) //Вибачайте, але прийдеться через костилі

	externalCallFinished = make(chan ExternalTaskResult, 10000)

	nonvoidDatatypes = []string{
		"i64",
		"i32",
		"i16",
		"i8",
		"u64",
		"u32",
		"u16",
		"u8",

		"f64",
		"f32",

		"bool",
		"error",
		"pointer",
		"func",
		"struct",
		"string",
		"table",
	}
)

type ExternalTask struct {
	Addr uintptr

	ArgsValues [][]Node

	Inter *Interpreter

	FuncCall *FuncCall
}

type ExternalTaskResult struct {
	R1, R2 uintptr
	Error  error
}

type Scope struct {
	Interpreter    *Interpreter
	Data           map[any]*Cell
	Pointers       map[unsafe.Pointer]*Cell
	Parent         *Scope
	IsFunc, IsLoop bool
	ImportedLibs   []string
	MainScope      bool
}

type Cell struct {
	Int64         int64
	Int32         int32
	Int16         int16
	Int8          int8
	Uint8         uint8
	Uint16        uint16
	Uint32        uint32
	Uint64        uint64
	Float64       float64
	Float32       float32
	BoolValue     bool
	StringValue   string
	StructValue   *Structure
	InstanceValue *StructObject
	TableValue    *Map
	FuncValue     *FuncDec
	PtrValue      uintptr
	ErrorValue    error
	AnyValue      any

	Bits uint8 //Shouldn't be used anywhere except *Cell structure methods

	DataType string
	Ptr      unsafe.Pointer
	TempBuf  any

	Scope *Scope
}

func (cell *Cell) InitFromRaw(value any, dataType string, nonptr bool, x, y int) {
	cell.Clear()
	cell.DataType = dataType

	switch dataType {
	case "i8", "i16", "i32", "i64":
		bits := uint8(twoDigitStr(dataType[1:]))

		rawInt64 := checkType[rawint64](value)
		rawUint64 := checkType[rawuint64](value)
		if !rawInt64 && !rawUint64 && cell.DataType != getValueType(value) {
			throw(cell.Scope.Interpreter.CurrentFileName, "Type mismatch: expected '%s' got '%s'", x, y, cell.DataType, getValueType(value))
		}
		if !rawInt64 {
			value = rawint64(toInt64(value))
		}

		switch bits {
		case 8:
			cell.Set(int8(value.(rawint64)), false, x, y)
		case 16:
			cell.Set(int16(value.(rawint64)), false, x, y)
		case 32:
			cell.Set(int32(value.(rawint64)), false, x, y)
		case 64:
			cell.Set(int64(value.(rawint64)), false, x, y)
		}
	case "u8", "u16", "u32", "u64":
		bits := uint8(twoDigitStr(dataType[1:]))

		rawInt64 := checkType[rawint64](value)
		rawUint64 := checkType[rawuint64](value)
		if !rawInt64 && !rawUint64 && cell.DataType != getValueType(value) {
			throw(cell.Scope.Interpreter.CurrentFileName, "Type mismatch: expected '%s' got '%s'", x, y, cell.DataType, getValueType(value))
		}
		if !rawUint64 {
			value = rawuint64(toUint64(value))
		}

		switch bits {
		case 8:
			cell.Set(uint8(value.(rawuint64)), false, x, y)
		case 16:
			cell.Set(uint16(value.(rawuint64)), false, x, y)
		case 32:
			cell.Set(uint32(value.(rawuint64)), false, x, y)
		case 64:
			cell.Set(uint64(value.(rawuint64)), false, x, y)
		}
	case "f32", "f64":
		bits := uint8(twoDigitStr(dataType[1:]))

		if !checkType[float64](value) {
			throw(cell.Scope.Interpreter.CurrentFileName, "Type mismatch: expected '%s' got '%s'", x, y, cell.DataType, getValueType(value))
		}

		cell.Bits = bits

		if bits == 32 {
			cell.Set(float32(value.(float64)), false, x, y)
		} else {
			cell.Set(value.(float64), false, x, y)
		}
	case "bool":
		if !checkType[bool](value) {
			throw(cell.Scope.Interpreter.CurrentFileName, "Type mismatch: expected '%s' got '%s'", x, y, cell.DataType, getValueType(value))
		}

		cell.Set(value.(bool), false, x, y)
	case "pointer":
		if !checkType[uintptr](value) {
			throw(cell.Scope.Interpreter.CurrentFileName, "Type mismatch: expected '%s' got '%s'", x, y, cell.DataType, getValueType(value))
		}

		cell.Set(value.(uintptr), false, x, y)
	case "string":
		if !checkType[string](value) {
			throw(cell.Scope.Interpreter.CurrentFileName, "Type mismatch: expected '%s' got '%s'", x, y, cell.DataType, getValueType(value))
		}

		cell.Set(value.(string), false, x, y)
	case "func":
		if !checkType[*FuncDec](value) {
			throw(cell.Scope.Interpreter.CurrentFileName, "Type mismatch: expected '%s' got '%s'", x, y, cell.DataType, getValueType(value))
		}

		cell.Set(value.(*FuncDec), false, x, y)
	case "struct":
		if !checkType[*Structure](value) {
			throw(cell.Scope.Interpreter.CurrentFileName, "Type mismatch: expected '%s' got '%s'", x, y, cell.DataType, getValueType(value))
		}

		cell.Set(value.(*Structure), false, x, y)
	case "table":
		if !checkType[*Map](value) && value != nil {
			throw(cell.Scope.Interpreter.CurrentFileName, "Type mismatch: expected '%s' got '%s'", x, y, cell.DataType, getValueType(value))
		}

		cell.Set(value.(*Map), false, x, y)
	case "error":
		if !checkType[error](value) {
			throw(cell.Scope.Interpreter.CurrentFileName, "Type mismatch: expected '%s' got '%s'", x, y, cell.DataType, getValueType(value))
		}

		cell.Set(value.(error), false, x, y)
	case "void":
		if value != nil {
			throw(cell.Scope.Interpreter.CurrentFileName, "Type mismatch: expected '%s' got '%s'", x, y, cell.DataType, getValueType(value))
		}

		cell.Set(nil, false, x, y)
	case "any":
		switch v := value.(type) {
		case rawint64:
			value = int64(v)
		case rawuint64:
			value = uint64(v)
		}

		cell.Set(value, false, x, y)
	default:
		structureCell := cell.Scope.GetCell(cell.DataType)
		if structureCell == nil || structureCell.DataType != "struct" {
			throw(cell.Scope.Interpreter.CurrentFileName, "Unexisting type: '%s'", x, y, cell.DataType)
		}

		structObject, ok := value.(*StructObject)
		if !ok || structObject.Identifier != cell.DataType {
			throw(cell.Scope.Interpreter.CurrentFileName, "Type mismatch: expected '%s' got '%s'", x, y, cell.DataType, getValueType(value))
		}

		cell.Set(value, false, x, y)
	}
}

func (cell *Cell) Set(value any, nonptr bool, x, y int) {
	switch avalue := value.(type) {
	case rawint64:
		bits := twoDigitStr(cell.DataType[1:])
		if strings.HasPrefix(cell.DataType, "i") {
			value = toInt(int64(avalue), -bits)
		} else {
			value = toInt(int64(avalue), bits)
		}
	case rawuint64:
		bits := twoDigitStr(cell.DataType[1:])
		if strings.HasPrefix(cell.DataType, "i") {
			value = toInt(int64(avalue), -bits)
		} else {
			value = toUint(uint64(avalue), bits)
		}
	}

	if cell.DataType != "" && cell.DataType != "any" && getValueType(value) != cell.DataType && !(value == nil && !slices.Contains(nonvoidDatatypes, cell.DataType)) {
		throw(cell.Scope.Interpreter.CurrentFileName, "Type mismatch: expected '%s' got '%s'", x, y, cell.DataType, getValueType(value))
	}

	if cell.DataType == "" {
		cell.DataType = getValueType(value)
	}

	switch cell.DataType {
	case "i64":
		cell.Int64 = value.(int64)
		cell.Bits = 64
		if !nonptr {
			cell.Ptr = unsafe.Pointer(&cell.Int64)
		}
	case "i32":
		cell.Int32 = value.(int32)
		cell.Bits = 32
		if !nonptr {
			cell.Ptr = unsafe.Pointer(&cell.Int32)
		}
	case "i16":
		cell.Int16 = value.(int16)
		cell.Bits = 16
		if !nonptr {
			cell.Ptr = unsafe.Pointer(&cell.Int16)
		}
	case "i8":
		cell.Int8 = value.(int8)
		cell.Bits = 8
		if !nonptr {
			cell.Ptr = unsafe.Pointer(&cell.Int8)
		}
	case "u64":
		cell.Uint64 = value.(uint64)
		cell.Bits = 64
		if !nonptr {
			cell.Ptr = unsafe.Pointer(&cell.Uint64)
		}
	case "u32":
		cell.Uint32 = value.(uint32)
		cell.DataType = "u32"
		cell.Bits = 32
		if !nonptr {
			cell.Ptr = unsafe.Pointer(&cell.Uint32)
		}
	case "u16":
		cell.Uint16 = value.(uint16)
		cell.Bits = 16
		if !nonptr {
			cell.Ptr = unsafe.Pointer(&cell.Uint16)
		}
	case "u8":
		cell.Uint8 = value.(uint8)
		cell.Bits = 8
		if !nonptr {
			cell.Ptr = unsafe.Pointer(&cell.Uint8)
		}
	case "f64":
		cell.Float64 = value.(float64)
		cell.Bits = 64
		if !nonptr {
			cell.Ptr = unsafe.Pointer(&cell.Float64)
		}
	case "f32":
		cell.Float32 = value.(float32)
		cell.Bits = 32
		if !nonptr {
			cell.Ptr = unsafe.Pointer(&cell.Float32)
		}
	case "bool":
		cell.BoolValue = value.(bool)
		if !nonptr {
			cell.Ptr = unsafe.Pointer(&cell.BoolValue)
		}
	case "string":
		cell.StringValue = value.(string)

		ptr, buf := valueToPtr(cell.Scope.Interpreter, cell.StringValue, 0, 0)
		cell.TempBuf = buf

		if !nonptr {
			cell.Ptr = unsafe.Pointer(ptr)
		}
	case "struct":
		structure := value.(*Structure)

		cell.StructValue = structure

		if !nonptr {
			cell.Ptr = unsafe.Pointer(structure)
		}
	case "func":
		function := value.(*FuncDec)

		cell.Bits = 0
		cell.FuncValue = function

		if !nonptr {
			cell.Ptr = unsafe.Pointer(function)
		}
	case "table":
		table := value.(*Map)

		cell.Bits = 0
		cell.TableValue = table

		table.ToMemory()

		if !nonptr {
			cell.Ptr = unsafe.Pointer(table.Address())
		}
	case "pointer":
		cell.Bits = 64
		cell.PtrValue = value.(uintptr)

		if !nonptr {
			cell.Ptr = unsafe.Pointer(&cell.PtrValue)
		}
	case "error":
		cell.ErrorValue = value.(error)

		if !nonptr {
			cell.Ptr = unsafe.Pointer(&cell.ErrorValue)
		}
	case "any":
		cell.AnyValue = value

		if !nonptr {
			cell.Ptr = unsafe.Pointer(&cell.AnyValue)
		}
	case "void":
		cell.Clear()
	default:
		instance, ok := value.(*StructObject)
		if !ok {
			panic("Unsupported type: " + fmt.Sprintf("'%s'", getValueType(value)))
		}

		cell.InstanceValue = instance

		if !nonptr {
			cell.Ptr = unsafe.Pointer(instance.Address())
		}
	}
}

func (cell *Cell) Clear() {
	cell.Bits = 0
	cell.BoolValue = false
	cell.ErrorValue = nil
	cell.Float32 = 0
	cell.Float64 = 0
	cell.FuncValue = nil
	cell.InstanceValue = nil
	cell.Int8 = 0
	cell.Int16 = 0
	cell.Int32 = 0
	cell.Int64 = 0
	cell.Uint8 = 0
	cell.Uint16 = 0
	cell.Uint32 = 0
	cell.Uint64 = 0
	cell.PtrValue = 0
	cell.StringValue = ""
	cell.StructValue = nil
	cell.TableValue = nil
	cell.TempBuf = nil
	cell.AnyValue = nil

	cell.Ptr = nil

	cell.DataType = ""
}

func (cell *Cell) Get() any {
	switch cell.DataType {
	case "i8", "i16", "i32", "i64":
		bits := cell.Bits

		switch bits {
		case 8:
			return cell.Int8
		case 16:
			return cell.Int16
		case 32:
			return cell.Int32
		case 64:
			return cell.Int64
		}
	case "u8", "u16", "u32", "u64":
		bits := cell.Bits

		switch bits {
		case 8:
			return cell.Uint8
		case 16:
			return cell.Uint16
		case 32:
			return cell.Uint32
		case 64:
			return cell.Uint64
		}
	case "f32", "f64":
		if cell.Bits == 32 {
			return cell.Float32
		}
		return cell.Float64
	case "bool":
		return cell.BoolValue
	case "string":
		return cell.StringValue
	case "struct":
		return cell.StructValue
	case "table":
		return cell.TableValue
	case "pointer":
		return cell.PtrValue
	case "func":
		return cell.FuncValue
	case "error":
		return cell.ErrorValue
	case "any":
		return cell.AnyValue
	case "nil":
		return nil
	default:
		return cell.InstanceValue
	}
	panic("Idk")
}

func (cell *Cell) GetAddress() unsafe.Pointer {
	return cell.Ptr
}

func CLPTR(scope *Scope, dataType string, v any, x, y int) *Cell {
	cell := &Cell{
		Scope: scope,
	}

	cell.InitFromRaw(v, dataType, false, x, y)

	return cell
}

func checkType[T any](v any) bool {
	_, ok := v.(T)
	return ok
}

type Map struct {
	*orderedmap.OrderedMap[any, *Cell]

	DataType string

	Pointers []any
	Layout   []string
	Mem      []byte
}

func anyToBytes(v []any, m *Map) []byte {
	buf := new(bytes.Buffer)

	m.Layout = make([]string, len(v))
	m.Pointers = make([]any, len(v))

	for i, x := range v {
		switch t := x.(type) {
		case int64, int32, int16, int8, uint8, uint16, uint32, uint64:
			m.Layout[i] = getValueType(t)
			m.Pointers[i] = nil

			binary.Write(buf, binary.LittleEndian, t)
		case float64, float32:
			m.Layout[i] = getValueType(t)
			m.Pointers[i] = nil

			binary.Write(buf, binary.LittleEndian, t)
		case bool:
			m.Layout[i] = getValueType(t)
			m.Pointers[i] = nil

			binary.Write(buf, binary.LittleEndian, t)
		case string:
			m.Layout[i] = getValueType(t)
			m.Pointers[i] = nil

			binary.Write(buf, binary.LittleEndian, append([]byte(t), 0))
		case error:
			m.Layout[i] = getValueType(t)
			m.Pointers[i] = nil

			binary.Write(buf, binary.LittleEndian, append([]byte(t.Error()), 0))
		case uintptr:
			m.Layout[i] = getValueType(t)
			m.Pointers[i] = nil

			binary.Write(buf, binary.LittleEndian, uint64(t))
		case unsafe.Pointer:
			m.Layout[i] = getValueType(t)
			m.Pointers[i] = nil

			binary.Write(buf, binary.LittleEndian, uint64(uintptr(t)))
		case *Map:
			m.Layout[i] = getValueType(t)
			m.Pointers[i] = t

			binary.Write(buf, binary.LittleEndian, uint32(len(t.Mem)))
			buf.Write(t.Mem)
		case nil:
			m.Layout[i] = getValueType(t)
			m.Pointers[i] = nil

			binary.Write(buf, binary.LittleEndian, 0)
		case *StructObject:
			m.Layout[i] = "instance"
			m.Pointers[i] = t

			binary.Write(buf, binary.LittleEndian, uint32(len(t.LastMem)))
			buf.Write(t.LastMem)
		default:
			fmt.Printf("%T\n", t)
			panic("Unsupported type")
		}
	}
	return buf.Bytes()
}

func bytesToAny(mem []byte, layout []string, pointers []any) []any {
	res := make([]any, len(layout))
	r := bytes.NewReader(mem)

	for i, t := range layout {
		switch t {
		case "table":
			var ln uint32
			binary.Read(r, binary.LittleEndian, &ln)

			b := make([]byte, ln)
			_, err := r.Read(b)
			if err != nil {
				panic("Failed to read table bytes")
			}

			m, ok := pointers[i].(*Map)
			if !ok {
				panic("Failed to find table's pointer")
			}

			m.Mem = b
			res[i] = m
		case "i64":
			var v int64
			binary.Read(r, binary.LittleEndian, &v)

			res[i] = v
		case "i32":
			var v int32
			binary.Read(r, binary.LittleEndian, &v)

			res[i] = v
		case "i16":
			var v int16
			binary.Read(r, binary.LittleEndian, &v)

			res[i] = v
		case "i8":
			var v int8
			binary.Read(r, binary.LittleEndian, &v)

			res[i] = v
		//Unsigned!
		case "u32":
			var v uint32
			binary.Read(r, binary.LittleEndian, &v)

			res[i] = v
		case "u16":
			var v uint16
			binary.Read(r, binary.LittleEndian, &v)

			res[i] = v
		case "u8":
			var v uint8
			binary.Read(r, binary.LittleEndian, &v)

			res[i] = v
		case "u64":
			var v uint64
			binary.Read(r, binary.LittleEndian, &v)

			res[i] = v
		case "pointer":
			var v uint64
			binary.Read(r, binary.LittleEndian, &v)

			res[i] = uintptr(v)
		//Unsigned end!
		case "f64":
			var v float64
			binary.Read(r, binary.LittleEndian, &v)

			res[i] = v
		case "f32":
			var v float32
			binary.Read(r, binary.LittleEndian, &v)

			res[i] = v
		case "bool":
			var b byte
			binary.Read(r, binary.LittleEndian, &b)

			res[i] = b != 0
		case "string":
			var ln uint32
			handle(binary.Read(r, binary.LittleEndian, &ln))

			b := make([]byte, ln)
			_, err := r.Read(b)
			handle(err)

			res[i] = string(b)
		case "error":
			var ln uint32
			handle(binary.Read(r, binary.LittleEndian, &ln))

			b := make([]byte, ln)
			_, err := r.Read(b)
			handle(err)

			res[i] = errors.New(string(b))
		default:
			panic("Unsupported type: " + t)
		}
	}

	return res
}

func (m *Map) ToMemory() {
	arrayBytes := anyToBytes(mapToSliceAny(m), m)
	if len(arrayBytes) > len(m.Mem) {
		m.Mem = arrayBytes
	} else {
		copy(m.Mem, arrayBytes)
	}
}

func (m *Map) FromMemory(x, y int) {
	s := bytesToAny(m.Mem, m.Layout, m.Pointers)
	//ASDDADS
	i := 0
	for _, value := range m.AllFromFront() {
		value.Set(s[i], false, x, y)
		i++
	}
}

func (m *Map) Address() uintptr {
	if len(m.Mem) == 0 {
		return 0
	}

	return uintptr(unsafe.Pointer(&m.Mem[0]))
}

var (
	filesBeingUsed = [][2]string{}

	osTags = []string{
		"_" + runtime.GOOS,
		"",
	}

	typeName = regexp.MustCompile(`<\*(?:[^.]+\.)?([^ ]+)`)
)

func NewScope(inter *Interpreter, parent *Scope) *Scope {
	return &Scope{
		Interpreter: inter,
		Data:        make(map[any]*Cell),
		Pointers:    make(map[unsafe.Pointer]*Cell),
		Parent:      parent,
	}
}

func ptrToAny(v any) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&v))
}

func (scope *Scope) Add(key, value any, dataType string, x, y int) (success bool) {
	if key == "_" {
		return true
	}
	cell := &Cell{
		Scope: scope,
	}
	cell.InitFromRaw(value, dataType, false, x, y)

	scope.Data[key] = cell
	scope.Pointers[cell.Ptr] = cell
	switch value := value.(type) {
	case *StructObject:
		for _, field := range value.Fields {
			fcell := field.Value

			scope.Pointers[fcell.Ptr] = fcell
		}
		for _, field := range value.Methods {
			fcell := field.Func

			scope.Pointers[fcell.Ptr] = fcell
		}
	case *Map:
		for _, vcell := range value.AllFromFront() {
			scope.Pointers[vcell.Ptr] = vcell
		}
	}

	return true
}

func (scope *Scope) Set(key, value any, x, y int) (success bool) {
	if key == "_" {
		return true
	}
	if oldvalue, ok := scope.Data[key]; ok {
		switch oldvalue.Get().(type) {
		case *Structure, *FuncDec:
			throw(scope.Interpreter.CurrentFileName, "Assignment to non-variable value", x, y)
		}

		cell := scope.Data[key]
		if cell.Ptr != nil {
			delete(scope.Pointers, cell.Ptr)
		}
		cell.Set(value, false, x, y)

		return true
	} else if scope.Parent != nil {
		return scope.Parent.Set(key, value, x, y)
	}
	return false
}

func (scope *Scope) Get(key any) (any, bool) {
	v, ok := scope.Data[key]
	if ok {
		return v.Get(), true
	} else if scope.Parent != nil {
		return scope.Parent.Get(key)
	}
	return nil, false
}

func (scope *Scope) GetWithAddress(ptr unsafe.Pointer) any {
	v, ok := scope.Pointers[ptr]
	if ok {
		return v.Get()
	} else if scope.Parent != nil {
		return scope.Parent.GetWithAddress(ptr)
	}
	return nil
}

func (scope *Scope) GetCellWithAddress(ptr unsafe.Pointer) *Cell {
	v, ok := scope.Pointers[ptr]
	if ok {

		return v
	} else if scope.Parent != nil {
		return scope.Parent.GetCellWithAddress(ptr)
	}
	return nil
}

func (scope *Scope) GetCell(key any) *Cell {
	v, ok := scope.Data[key]
	if ok {
		return v
	} else if scope.Parent != nil {
		return scope.Parent.GetCell(key)
	}
	return nil
}

type Structure struct {
	Identifier string
	Fields     []*FieldDecl
}

func (structure *Structure) CheckField(name string) bool {
	for _, field := range structure.Fields {
		if field.Identifier == name {
			return true
		}
	}
	return false
}

func (structure *Structure) GetField(name string) *FieldDecl {
	for _, field := range structure.Fields {
		if field.Identifier == name {
			return field
		}
	}
	return nil
}

func (structure *Structure) IsAFunc(name string) bool {
	for _, field := range structure.Fields {
		if field.Identifier == name {
			return field.Func != nil
		}
	}
	return false
}

func (structure *Structure) CountMethods() int {
	count := 0
	for _, field := range structure.Fields {
		if field.Func != nil {
			count++
		}
	}
	return count
}

type FieldDecl struct {
	Identifier, DataType string
	Method               bool
	Func                 *FuncDec
}

func newInstance(name string, fields []*Field, methods []*Method) *StructObject {
	return &StructObject{
		Identifier: name,
		Fields:     fields,
		Methods:    methods,
		LastMem:    []byte{},
	}
}

type StructObject struct {
	scope      *Scope
	Identifier string
	Fields     []*Field
	Methods    []*Method
	LastMem    []byte
}

type Field struct {
	Identifier, DataType string
	Value                *Cell
}

type Method struct {
	Identifier string
	Func       *Cell
}

type FieldLayout struct {
	Name   string
	Offset uintptr
	Size   uintptr
	Type   string // например "uint32", "uintptr"
}

func (s *StructObject) ToMemoryLayout(layout []FieldLayout) []byte {
	var mem []byte
	if len(layout) == 0 {
		s.LastMem = mem
		return mem
	}

	size := layout[len(layout)-1].Offset + layout[len(layout)-1].Size

	if len(s.LastMem) == 0 {
		mem = make([]byte, size)
	} else if len(s.LastMem) < int(size) {
		newMem := make([]byte, size)

		copy(newMem, s.LastMem)

		mem = newMem
	} else {
		mem = s.LastMem
	}

	for _, lf := range layout {
		val, ok := s.GetCell(lf.Name)
		if !ok {
			continue
		}

		offset := int(lf.Offset)
		switch lf.Type {
		case "i8":
			v := int8(toInt64(val.Get()))
			mem[offset] = byte(v)

		case "u8":
			v := uint8(toUint64(val.Get()))
			mem[offset] = byte(v)

		case "i16":
			v := int16(toInt64(val.Get()))
			binary.LittleEndian.PutUint16(mem[offset:], uint16(v))

		case "u16":
			v := uint16(toInt64(val.Get()))
			binary.LittleEndian.PutUint16(mem[offset:], v)

		case "i32":
			v := int32(toInt64(val.Get()))
			binary.LittleEndian.PutUint32(mem[offset:], uint32(v))

		case "u32":
			v := uint32(toInt64(val.Get()))
			binary.LittleEndian.PutUint32(mem[offset:], v)

		case "i64":
			v := int64(toInt64(val.Get()))
			binary.LittleEndian.PutUint64(mem[offset:], toUint64(v))

		case "u64", "ptr":
			v := toUint64(val.Get())
			binary.LittleEndian.PutUint64(mem[offset:], v)

		case "f64":
			binary.LittleEndian.PutUint64(mem[offset:], math.Float64bits(val.Get().(float64)))
		case "f32":
			binary.LittleEndian.PutUint32(mem[offset:], math.Float32bits(val.Get().(float32)))
		case "bool":
			mem[offset] = byte(toUint64(val.Get()))
		case "instance":
			//binary.LittleEndian.PutUint64(mem[offset:], uint64(uintptr(val.Ptr)))
			instance := val.Get().(*StructObject)

			copy(mem[offset:], instance.LastMem)

		default:
			panic("Unsupported field type " + lf.Type)
		}
	}

	s.LastMem = mem
	return mem
}

func (s *StructObject) FromMemoryLayout(layout []FieldLayout, x, y int) {
	if s.LastMem == nil {
		return
	}
	mem := s.LastMem

	for _, lf := range layout {
		offset := int(lf.Offset)

		switch lf.Type {
		case "i8":
			v := int8(mem[offset])
			s.Set(lf.Name, int64(v), x, y)

		case "u8":
			v := mem[offset]
			s.Set(lf.Name, int64(v), x, y)

		case "i16":
			v := int16(binary.LittleEndian.Uint16(mem[offset:]))
			s.Set(lf.Name, int64(v), x, y)

		case "u16":
			v := binary.LittleEndian.Uint16(mem[offset:])
			s.Set(lf.Name, int64(v), x, y)

		case "i32":
			v := int32(binary.LittleEndian.Uint32(mem[offset:]))
			s.Set(lf.Name, int64(v), x, y)

		case "u32":
			v := binary.LittleEndian.Uint32(mem[offset:])
			s.Set(lf.Name, int64(v), x, y)

		case "i64":
			v := int64(binary.LittleEndian.Uint64(mem[offset:]))
			s.Set(lf.Name, v, x, y)

		case "u64", "ptr":
			v := binary.LittleEndian.Uint64(mem[offset:])
			s.Set(lf.Name, uintptr(v), x, y)

		case "f64":
			bits := binary.LittleEndian.Uint64(mem[offset:])
			v := math.Float64frombits(bits)
			s.Set(lf.Name, v, x, y)
		case "f32":
			bits := binary.LittleEndian.Uint32(mem[offset:])
			v := math.Float32frombits(bits)
			s.Set(lf.Name, v, x, y)
		case "bool":
			v := mem[offset]
			s.Set(lf.Name, v == 1, x, y)
		/*case "instance":
		ptr := binary.LittleEndian.Uint64(mem[offset:])
		if ptr == 0 {
			s.Set(lf.Name, nil)
			continue
		}

		val, ok := s.Get(lf.Name)
		if !ok || val == nil {
			continue
		}
		sub := val.(*StructObject)
		subLayout := sub.Layout()
		subSize := subLayout[len(subLayout)-1].Offset + subLayout[len(subLayout)-1].Size

		subMem := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), subSize)
		sub.LastMem = append([]byte(nil), subMem...)
		sub.FromMemoryLayout(subLayout)*/
		case "instance":
			val, ok := s.Get(lf.Name)
			if !ok || val == nil {
				continue
			}
			sub := val.(*StructObject)
			subLayout := sub.Layout()
			subSize := subLayout[len(subLayout)-1].Offset + subLayout[len(subLayout)-1].Size

			if offset+int(subSize) > len(mem) {
				panic("memory slice out of bounds")
			}
			subMem := make([]byte, subSize)
			copy(subMem, mem[offset:offset+int(subSize)])

			sub.LastMem = subMem
			sub.FromMemoryLayout(subLayout, x, y)
		default:
			panic("Unsupported field type " + lf.Type)
		}
	}
}

func (s *StructObject) Address() uintptr {
	if len(s.LastMem) == 0 {
		return 0
	}
	return uintptr(unsafe.Pointer(&s.LastMem[0]))
}

func toInt(v int64, bits int) any {
	switch bits {
	case 8:
		return uint8(v)
	case 16:
		return uint16(v)
	case 32:
		return uint32(v)
	case 64:
		return uint64(v)
	case -8:
		return int8(v)
	case -16:
		return int16(v)
	case -32:
		return int32(v)
	case -64:
		return int64(v)
	default:
		return v
	}
}

func toUint(v uint64, bits int) any {
	switch bits {
	case 8:
		return uint8(v)
	case 16:
		return uint16(v)
	case 32:
		return uint32(v)
	case 64:
		return uint64(v)
	case -8:
		return int8(v)
	case -16:
		return int16(v)
	case -32:
		return int32(v)
	case -64:
		return int64(v)
	default:
		return v
	}
}

func ftoInt(v float64, bits int) any {
	switch bits {
	case 8:
		return int8(v)
	case 16:
		return int16(v)
	case 32:
		return int32(v)
	case 64:
		return int64(v)
	default:
		return v
	}
}

func toInt64(v any) int64 {
	switch val := v.(type) {
	case int32:
		return int64(val)
	case uint32:
		return int64(val)
	case int64:
		return val
	case uint64:
		return int64(val)
	case int16:
		return int64(val)
	case uint16:
		return int64(val)
	case int8:
		return int64(val)
	case uint8:
		return int64(val)
	case rawint64:
		return int64(val)
	case rawuint64:
		return int64(val)
	case unsafe.Pointer:
		return int64(uintptr(val))
	case float64:
		return int64(val)
	case uintptr:
		return int64(val)
	default:
		return 0
	}
}

func toUint64(v any) uint64 {
	switch val := v.(type) {
	case int8:
		return uint64(val)
	case uint8:
		return uint64(val)
	case int16:
		return uint64(val)
	case uint16:
		return uint64(val)
	case int32:
		return uint64(val)
	case uint32:
		return uint64(val)
	case int64:
		return uint64(val)
	case rawint64:
		return uint64(val)
	case rawuint64:
		return uint64(val)
	case uint64:
		return val
	case unsafe.Pointer:
		return uint64(uintptr(val))
	case float64:
		return uint64(val)
	case uintptr:
		return uint64(val)
	case bool:
		if val {
			return 1
		}
		return 0
	default:
		return 0
	}
}

func alignf(offset, alignment uintptr) uintptr {
	if offset%alignment == 0 {
		return offset
	}
	return offset + alignment - (offset % alignment)
}

func (s *StructObject) Layout() []FieldLayout {
	layout := make([]FieldLayout, len(s.Fields))
	offset := uintptr(0)

	i := 0
	for _, field := range s.Fields {
		var size, align uintptr
		var typ string
		cell := field.Value

		switch field.DataType {
		case "i8", "u8":
			size, align = 1, 1
		case "i16", "u16":
			size, align = 2, 2
		case "i32", "u32":
			size, align = 4, 4
		case "i64", "u64":
			size, align = 8, 8
		case "f64":
			size, align = 8, 8
		case "f32":
			size, align = 4, 4
		case "pointer":
			size, align, typ = 8, 8, "ptr"
		case "bool":
			size, align, typ = 1, 1, "bool"
		default:
			switch v := cell.Get().(type) {
			case *StructObject:
				sub := v.Layout()
				last := sub[len(sub)-1]
				size = last.Offset + last.Size

				align, typ = 8, "instance"
			}
		}

		if typ == "" {
			typ = field.DataType
		}

		offset = alignf(offset, align)

		layout[i] = FieldLayout{
			Name:   field.Identifier,
			Offset: offset,
			Size:   size,
			Type:   typ,
		}

		offset += size
		i++
	}

	return layout
}

func (structObj *StructObject) Get(fieldName string) (any, bool) {
	for _, field := range structObj.Fields {
		if field.Identifier == fieldName {
			cell := field.Value
			return cell.Get(), true
		}
	}
	for _, field := range structObj.Methods {
		if field.Identifier == fieldName {
			cell := field.Func
			return cell.Get(), true
		}
	}
	return nil, false
}

func (structObj *StructObject) GetCell(fieldName string) (*Cell, bool) {
	for _, field := range structObj.Fields {
		if field.Identifier == fieldName {
			cell := field.Value
			return cell, true
		}
	}
	for _, method := range structObj.Methods {
		if method.Identifier == fieldName {
			cell := method.Func
			return cell, true
		}
	}
	return nil, false
}

func (structObj *StructObject) CheckFormat(format ...[2]string) bool {
	for _, format_i := range format {
		fieldName := format_i[0]
		fieldType := format_i[1]

		val, ok := structObj.Get(fieldName)
		if !ok {
			return false
		}
		if !checkDataType(fieldType, val) {
			return false
		}
	}
	return true
}

func (structObj *StructObject) Set(fieldName string, value any, x, y int) bool {
	for _, field := range structObj.Fields {
		cell := field.Value
		if field.Identifier == fieldName {
			cell.Set(value, false, x, y)

			return true
		}
	}

	for _, method := range structObj.Methods {
		if method.Identifier == fieldName {
			funcDecl := method.Func.Get().(*FuncDec)
			throw(structObj.scope.Interpreter.CurrentFileName, "Cannot assign value to a instance's method.", funcDecl.X, funcDecl.Y)
		}
	}
	return false
}

func format(v ...any) string {
	var formated string

	for i, a := range v {
		var suffix string
		if i != len(v)-1 {
			suffix = " "
		}

		switch a := a.(type) {
		case *Map:
			mapFormat := "%s{%s}"
			elemFormat := "[%s]: %s,"

			var elements string
			i := 0
			for k, v := range a.AllFromFront() {
				var elementSuffix string
				if i != a.Len()-1 {
					elementSuffix = " "
				}

				elements += fmt.Sprintf(elemFormat+elementSuffix, format(k), format(v.Get()))
				i++
			}

			formated += fmt.Sprintf(mapFormat, a.DataType, elements) + suffix
		case error:
			formated += fmt.Sprint(a.Error()) + suffix
		case *FuncDec, *Structure:
			formated += fmt.Sprintf("%p", a) + suffix
		case *StructObject:
			if a == nil {
				return formated + format(nil) + suffix
			}

			structFormat := "%s{%s}"
			fieldFormat := "%s: %s;"

			var fields string

			i := 0
			for _, field := range a.Fields {
				var fieldSuffix string
				if i != len(a.Fields)-1 {
					fieldSuffix = " "
				}
				cell := field.Value

				fields += fmt.Sprintf(fieldFormat+fieldSuffix, field.Identifier, format(cell.Get()))
				i++
			}

			formated += fmt.Sprintf(structFormat, a.Identifier, fields) + suffix
		case uintptr:
			formated += fmt.Sprintf("%#x", a) + suffix
		case nil:
			for k, t := range tokenTypes {
				if t == "nil" {
					formated += fmt.Sprintf("<%s>", k) + suffix
					break
				}
			}
		default:
			formated += fmt.Sprint(a) + suffix
		}
	}

	return formated
}

func importModule(path string, mainScope *Scope, x, y int) {
	if !strings.HasSuffix(path, fileType) {
		path += fileType
	}
	pathNS, _ := strings.CutSuffix(path, fileType)
	absPath := getAbsPath(path)

	for _, tag := range osTags {
		absPathTag := absPath + tag
		var finalPath string

		for _, filePath := range filesBeingUsed {
			if filePath[0] == absPathTag {
				throwNoPos("Recursive or duplicate import of file '%s' detected.", filePath[1])
			}
		}

		finalPath = pathNS + tag + fileType
		_, err := os.Stat(finalPath)
		if err != nil {
			if os.IsNotExist(err) {
				finalPath = filepath.Join(libs, pathNS) + tag + fileType

				_, errS := os.Stat(finalPath)
				if errS != nil {
					continue
				}
			} else {
				continue
			}
		}

		moduleData := run(finalPath, path, false)

		for k, v := range moduleData {
			cell := &Cell{
				Scope: mainScope,
			}
			cell.Set(v.Get(), false, x, y)

			mainScope.Data[k] = cell
			mainScope.Pointers[cell.Ptr] = cell
		}
		return
	}
	throwNoPos("Invalid file or library '%s'", path)
}

func mapToSliceAny(m *Map) []any {
	slice := make([]any, m.Len())

	i := 0
	for _, v := range m.AllFromFront() {
		slice[i] = v.Get()
		i++
	}

	return slice
}

func getInterfaceType(v any) string {
	typeSM := typeName.FindAllStringSubmatch(reflect.ValueOf(v).String(), -1)
	if len(typeSM) == 0 {
		return ""
	}

	return typeSM[0][1]
}

type Interpreter struct {
	CurrentFileName string
	AST             []Node
	CurrentScope    *Scope
	UnableToImport  bool
}

func NewInterpreter(filename string, ast []Node) *Interpreter {
	return &Interpreter{
		CurrentFileName: filename,
		AST:             ast,
	}
}

func (inter *Interpreter) GetBinOpValue(node *BinOpNode) any {
	if node.operator == "sub" && node.L == nil && node.R != nil {
		value := inter.GetNodeValue(node.R)
		/*if checkType[rawint64](value) {
			value = int64(value.(rawint64))
		}*/

		rtype := checkDataType("number", value)
		if rtype || checkType[rawint64](value) || checkType[rawuint64](value) {
			switch value := value.(type) {
			case rawuint64:
				return -value
			case rawint64:
				return -value
			case int64:
				return -value
			case int32:
				return -value
			case int16:
				return -value
			case int8:
				return -value
			case uint64:
				return -value
			case uint32:
				return -value
			case uint16:
				return -value
			case uint8:
				return -value
			case float64:
				return -value
			case float32:
				return -value
			}
		}
		fmt.Printf("%T\n", value)
		throw(inter.CurrentFileName, "Unable to use unary operator '-' on non-number value.", node.X, node.Y)
	}

	err := "Cannot perform binary operations on multiple values at the same time."

	l, r := inter.GetNodeValue(node.L), inter.GetNodeValue(node.R)

	returnL, ok := l.([]any)
	if ok {
		if len(returnL) > 1 {
			throw(inter.CurrentFileName, err, node.X, node.Y)
		}
		l = returnL[0]
	}

	returnR, ok := r.([]any)
	if ok {
		if len(returnR) > 1 {
			throw(inter.CurrentFileName, err, node.X, node.Y)
		}
		r = returnR[0]
	}

	switch node.operator {
	case "and":
		return l == true && r == true
	case "or":
		return l == true || r == true
	}

	f := binOperations[node.operator]

	if checkType[rawint64](l) {
		l = int64(l.(rawint64))
	}
	if checkType[rawint64](r) {
		r = int64(r.(rawint64))
	}

	if checkType[rawuint64](l) {
		l = uint64(l.(rawuint64))
	}
	if checkType[rawuint64](r) {
		r = uint64(r.(rawuint64))
	}

	return f(inter, l, r, node.X, node.Y)
}

func (inter *Interpreter) GetNodeValue(node Node) any {
	switch node := node.(type) {
	case *NilNode:
		return nil
	case *KeyNilNode:
		return node
	case *IntNode:
		if node.ValueI64 == 0 && node.ValueU64 > 0 {
			return node.ValueU64
		} else {
			return node.ValueI64
		}
	case *FloatNode:
		return node.Value
	case *StrNode:
		return node.Value
	case *BoolNode:
		return node.Value
	case *BinOpNode:
		return inter.GetBinOpValue(node)
	case *TypeAssert:
		target := inter.GetNodeValue(node.Target)
		typeName := node.Type.Value

		assertValue, ok := assertType(target, typeName)
		if !ok {
			throw(inter.CurrentFileName, "Error occured while tried to assert value type of '%s' to '%s'", node.X, node.Y, getValueType(target), typeName)
		}

		return assertValue
	case *Brackets:
		return inter.GetNodeValueS(node.Value, node.X, node.Y)
	case *MapNode:
		return inter.GetMap(node)
	case *FuncDec:
		return node
	case *FuncCall:
		return inter.CallFunction(node)
	case *IdentNode:
		v, found := inter.CurrentScope.Get(node.Value)
		if !found {
			throw(inter.CurrentFileName, "Variable '%s' doesn't exist", node.X, node.Y, node.Value)
		}

		return v
	case *StructNode:
		return inter.NewStructObject(node)
	case *GetPtrNode:
		if node.Src == nil {
			throw(inter.CurrentFileName, "Attempt to get a pointer of nothing.", node.X, node.Y)
		}
		srcNode := node.Src

		scope := inter.CurrentScope

		switch srcNode := srcNode.(type) {
		case *IdentNode:
			identifier := srcNode.Value

			cell := scope.GetCell(identifier)
			if cell == nil {
				throw(inter.CurrentFileName, "Attempt to get a pointer of non-existing value.", node.X, node.Y)
			}
			if cell.Ptr == nil {
				throw(inter.CurrentFileName, "Attempt to get pointer of nil value.", node.X, node.Y)
			}

			switch v := cell.Get().(type) {
			case *FuncDec, *Structure:
				throw(inter.CurrentFileName, "Cannot get a pointer of '%s' value", node.X, node.Y, getValueType(v))
			}

			return uintptr(cell.Ptr) //uintptr(unsafe.Pointer(&cell.Ptr))
		case *GetElementNode:
			tableNode, keyNodes := inter.GetTableAndKeys(srcNode, []Node{})
			if tableNode == nil {
				throw(inter.CurrentFileName, "Attempt to index nothing.", srcNode.X, srcNode.Y)
			}

			table := inter.GetNodeValue(tableNode)

			keys := make([]any, len(keyNodes))
			for i, keyNode := range keyNodes {
				keys[i] = inter.GetNodeValue(keyNode)
			}

			switch table := table.(type) {
			case *Map, string:
				cell := inter.GetTableCellByKeys(table, keys, srcNode, 0)

				return cell.Ptr
			default:
				throw(inter.CurrentFileName, "Cannot index non-table value.", node.X, node.Y)
			}
		case *GetFieldNode:
			cell := inter.GetInstanceFieldCell(srcNode)

			return cell.Ptr
		}
	case *GetFieldNode:
		cell := inter.GetInstanceFieldCell(node)

		return cell.Get()
	case *GetElementNode:
		tableNode, keyNodes := inter.GetTableAndKeys(node, []Node{})
		if tableNode == nil {
			throw(inter.CurrentFileName, "Attempt to index nothing.", node.X, node.Y)
		}

		table := inter.GetNodeValue(tableNode)

		keys := make([]any, len(keyNodes))
		for i, keyNode := range keyNodes {
			keys[i] = inter.GetNodeValue(keyNode)
		}

		switch table := table.(type) {
		case *Map, string:
			return inter.GetTableValueByKeys(table, keys, node, 0)
		default:
			throw(inter.CurrentFileName, "Cannot index non-table or non-string value.", node.X, node.Y)
		}
	}
	throw(inter.CurrentFileName, "Invalid node '%s'.", node.Position(), node.Line(), getInterfaceType(node))
	return nil
}

func (inter *Interpreter) GetNodeValueS(nodes []Node, x, y int) any {
	if len(nodes) > 1 || len(nodes) == 0 {
		throw(inter.CurrentFileName, "Value has more than one value or is empty", x, y)
	}
	return inter.GetNodeValue(nodes[0])
}

func (inter *Interpreter) GetTableAndKeys(node *GetElementNode, keys []Node) (Node, []Node) {
	if len(node.Map) > 1 {
		throw(inter.CurrentFileName, "Cannot index more than one value at the same time", node.X, node.Y)
	}
	if len(node.Key) > 1 {
		throw(inter.CurrentFileName, "Key cannot have more than one value", node.X, node.Y)
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

func (inter *Interpreter) GetTableValueByKeys(table any, keys []any, getElemN *GetElementNode, index int) any {
	if index >= len(keys) {
		return nil
	}

	key := keys[index]
	if checkType[rawint64](key) {
		key = int64(key.(rawint64))
	}
	if checkType[rawuint64](key) {
		key = uint64(key.(rawuint64))
	}

	tableCell, iscell := table.(*Cell)
	if iscell {
		table = tableCell.Get()
	}

	switch table := table.(type) {
	case *Map:
		if !table.Has(key) {
			if index+1 < len(keys) {
				throw(inter.CurrentFileName, "Attempt to index non-table value.", getElemN.X, getElemN.Y)
			} else {
				return nil
			}
		}

		elem := table.GetElement(key)
		var val any
		if elem != nil {
			val = elem.Value.Get()
		}

		if index+1 < len(keys) {
			return inter.GetTableValueByKeys(val, keys, getElemN, index+1)
		}
		return val
	case string:
		if !checkDataType("int", key) {
			throw(inter.CurrentFileName, "Attempt to index string with non-integer value; '%s'", getElemN.X, getElemN.Y, getValueType(key))
		}

		i := int(toInt64(key))
		if i >= len(table) || i < 0 {
			throw(inter.CurrentFileName, "Attempt to index a character beyond the string limit.", getElemN.X, getElemN.Y)
		}
		if index+1 != len(keys) {
			throw(inter.CurrentFileName, "Repeated indexing of a character is not allowed.", getElemN.X, getElemN.Y)
		}

		char := string([]rune(table)[i])

		return char
	}
	throw(inter.CurrentFileName, "Attempt to index non-table value.", getElemN.X, getElemN.Y)
	return nil
}

func (inter *Interpreter) GetTableCellByKeys(table any, keys []any, getElemN *GetElementNode, index int) *Cell {
	if index >= len(keys) {
		return nil
	}

	key := keys[index]
	if checkType[rawint64](key) {
		key = int64(key.(rawint64))
	}
	if checkType[rawuint64](key) {
		key = uint64(key.(rawuint64))
	}

	switch table := table.(type) {
	case *Map:
		if !table.Has(key) {
			if index+1 < len(keys) {
				throw(inter.CurrentFileName, "Attempt to index non-table value.", getElemN.X, getElemN.Y)
			} else {
				return CLPTR(inter.CurrentScope, "void", nil, getElemN.X, getElemN.Y)
			}
		}

		elem := table.GetElement(key)
		var val *Cell
		if elem != nil {
			val = elem.Value
		}

		if index+1 < len(keys) {
			return inter.GetTableCellByKeys(val.Get(), keys, getElemN, index+1)
		}
		return val
	}
	throw(inter.CurrentFileName, "Attempt to index non-table value.", getElemN.X, getElemN.Y)
	return nil
}

func (inter *Interpreter) GetStructAndFieldNames(node *GetFieldNode, fields []Node) (Node, []Node) {
	if len(node.Field) > 1 {
		throw(inter.CurrentFileName, "Cannot get value of more than one field at the same time.", node.X, node.Y)
	}
	fields = append(fields, node.Field[0])
	switch structNode := node.Struct.(type) {
	case *GetFieldNode:
		return inter.GetStructAndFieldNames(structNode, fields)
	default:
		slices.Reverse(fields)
		return structNode, fields
	}
}

func (inter *Interpreter) GetFieldValueByNames(structObj *StructObject, fieldNames []string, getFieldN *GetFieldNode, index int) any {
	if index >= len(fieldNames) {
		return nil
	}

	fieldName := fieldNames[index]

	val, ok := structObj.Get(fieldName)
	if !ok {
		throw(inter.CurrentFileName, "Attempt to get a value of nonexistent field '%s'", getFieldN.X, getFieldN.Y, fieldName)
	}

	if index+1 < len(fieldNames) {
		nextStructObj, ok := val.(*StructObject)
		if !ok {
			throw(inter.CurrentFileName, "Attempt to get field of a non-structure value", getFieldN.X, getFieldN.Y)
		}

		return inter.GetFieldValueByNames(nextStructObj, fieldNames, getFieldN, index+1)
	}
	return val
}

func (inter *Interpreter) GetFieldCellByNames(structObj *StructObject, fieldNames []string, getFieldN *GetFieldNode, index int) *Cell {
	if index >= len(fieldNames) {
		return nil
	}

	fieldName := fieldNames[index]

	val, ok := structObj.GetCell(fieldName)
	if !ok {
		throw(inter.CurrentFileName, "Attempt to get a value of nonexistent field '%s'", getFieldN.X, getFieldN.Y, fieldName)
	}

	if index+1 < len(fieldNames) {
		nextStructObj, ok := val.Get().(*StructObject)
		if !ok {
			throw(inter.CurrentFileName, "Attempt to get field of a non-structure value", getFieldN.X, getFieldN.Y)
		}

		return inter.GetFieldCellByNames(nextStructObj, fieldNames, getFieldN, index+1)
	}
	return val
}

func (inter *Interpreter) GetInstanceFieldCell(getFieldNode *GetFieldNode) *Cell {
	structObjNode, fieldNodes := inter.GetStructAndFieldNames(getFieldNode, []Node{})
	if structObjNode == nil {
		throw(inter.CurrentFileName, "Attempt to get field of nothing.", getFieldNode.X, getFieldNode.Y)
	}

	structObj, ok := inter.GetNodeValue(structObjNode).(*StructObject)
	if !ok {
		throw(inter.CurrentFileName, "Attempt to get field of a non-structure value.", structObjNode.Position(), structObjNode.Line())
	}

	fields := make([]string, len(fieldNodes))
	for i, fieldNode := range fieldNodes {

		fieldIdentNode, ok := fieldNode.(*IdentNode)
		if !ok {
			throw(inter.CurrentFileName, "Field name must be an identifier", fieldNode.Position(), fieldNode.Line())
		}

		fields[i] = fieldIdentNode.Value
	}

	return inter.GetFieldCellByNames(structObj, fields, getFieldNode, 0)
}

func (inter *Interpreter) GetMap(node *MapNode) *Map {
	m := orderedmap.NewOrderedMap[any, *Cell]()

	elemDataType := node.ElemDataType.Value

	for i, element := range node.Map {
		key, value := inter.GetNodeValueS(element.Key, element.X, element.Y), inter.GetNodeValueS(element.Value, element.X, element.Y)

		if checkType[*KeyNilNode](key) {
			key = int64(i)
		}

		values, ok := value.([]any)
		if ok {
			if len(values) > 1 {
				throw(inter.CurrentFileName, "Field cannot have more than one value.", element.X, element.Y)
			} else if len(values) == 0 {
				throw(inter.CurrentFileName, "Cannot assign a field cannot be an empty value.", element.X, element.Y)
			}

			m.Set(key, CLPTR(inter.CurrentScope, elemDataType, values[0], element.X, element.Y))
		} else {
			m.Set(key, CLPTR(inter.CurrentScope, elemDataType, value, element.X, element.Y))
		}
	}

	fmap := &Map{m, elemDataType, []any{}, []string{}, []byte{}}
	fmap.ToMemory()

	return fmap
}

func (inter *Interpreter) Current(scope *Scope) {
	inter.CurrentScope = scope
}

func (inter *Interpreter) CallFunction(node *FuncCall) []any {
	funcDecInterface := inter.GetNodeValue(node.Func)

	switch funcDec := funcDecInterface.(type) {
	case uintptr:
		argsValues := make([][]Node, len(node.Arguments))
		for i, argNode := range node.Arguments {
			argsValues[i] = []Node{argNode}
		}

		task := ExternalTask{
			Addr: funcDec,

			Inter: inter,

			ArgsValues: argsValues,

			FuncCall: node,
		}

		externalCallingChan <- task

		result := <-externalCallFinished

		return []any{result.R1, result.R2, error(result.Error)}
	case *FuncDec:
		if funcDec.Template != nil {
			args := []any{node.X, node.Y, inter}

			argsValues := make([][]Node, len(node.Arguments))
			for i, argNode := range node.Arguments {
				argsValues[i] = []Node{argNode}
			}

			cookedValues := inter.CookValues(
				uint(len(node.Arguments)),
				argsValues,
				node.X,
				node.Y,
			)

			for i, cookedValue := range cookedValues {
				switch cookedValue := cookedValue.(type) {
				case rawuint64:
					cookedValues[i] = uint64(cookedValue)
				case rawint64:
					cookedValues[i] = int64(cookedValue)
				}
			}

			result := funcDec.Template(
				append(args,
					cookedValues...,
				)...,
			)

			return result
		}

		body := funcDec.Body

		if len(node.Arguments) > len(funcDec.Arguments) {
			throw(inter.CurrentFileName, "Attempt to pass more arguments to a function call than function actually need.", node.X, node.Y)
		}

		argsIdentifiers := funcDec.Arguments //[]IdentNode{}
		argsValues := make([][]Node, len(node.Arguments))

		for i, argNode := range node.Arguments {
			argsValues[i] = []Node{argNode}
		}

		argsBody := []Node{
			&VarDec{
				Identifier: argsIdentifiers,
				Value:      argsValues,
				DataTypes:  funcDec.ArgumentsDataTypes,
				Argument:   true,
				X:          node.X,
				Y:          node.Y,
			},
		}

		addToScope := [][3]any{}

		body = slices.Concat(argsBody, body)
		if funcDec.Self != nil {
			addToScope = [][3]any{
				{selfKeyword, funcDec.Self, funcDec.Self.Identifier},
			}
		}

		_, _, value := inter.CompleteBody(body, true, false, addToScope...)

		return value
	default:
		throw(inter.CurrentFileName, "Attempt to call a non-function object.", node.X, node.Y)
		return nil
	}
}

func (inter *Interpreter) CompleteBody(body []Node, isFunc, isLoop bool, addToScope ...[3]any) (end, skip bool, value []any) {
	scope := NewScope(inter, inter.CurrentScope)
	scope.IsFunc = isFunc
	scope.IsLoop = isLoop

	inter.Current(scope)
	defer inter.Current(scope.Parent)

	for _, addToScopeElem := range addToScope {
		ident := addToScopeElem[0].(string)
		value := addToScopeElem[1]
		dataType := addToScopeElem[2].(string)

		scope.Add(ident, value, dataType, -1, -1)
	}

	for _, node := range body {
		end, skip, valuesAny := inter.CompleteNode(node)
		if len(valuesAny) > 0 || end || skip {
			var values []any = nil

			if len(valuesAny) > 0 {
				values = make([]any, 0, len(valuesAny))

				for _, value := range valuesAny {
					if (value == ReturnNil{}) {
						value = nil
					}
					values = append(values, value)
				}
			}
			return end, skip, values
		}
	}

	return false, false, nil
}

func (inter *Interpreter) SetTableElementValue(table *Map, keys []any, value any, index int, x, y int) {
	if index >= len(keys) {
		return
	}

	key := keys[index]

	elem := table.GetElement(key)
	if elem != nil {
		switch elem := elem.Value.Get().(type) {
		case *Map:
			if index+1 >= len(keys) {
				table.Set(key, CLPTR(inter.CurrentScope, table.DataType, value, x, y))

				elem.ToMemory()
				break
			}
			inter.SetTableElementValue(elem, keys, value, index+1, x, y)
			return
		}
	}
	table.Set(key, CLPTR(inter.CurrentScope, table.DataType, value, x, y))
	table.ToMemory()
}

func (inter *Interpreter) SetInstanceFieldValue(instance *StructObject, fields []string, value any, index int, x, y int) {
	if index >= len(fields) {
		return
	}

	field := fields[index]

	fieldVal, _ := instance.Get(field)
	switch fieldVal := fieldVal.(type) {
	case *StructObject:
		if index >= len(fields) {
			instance.Set(field, value, x, y)
			break
		}

		inter.SetInstanceFieldValue(fieldVal, fields, value, index+1, x, y)
	default:
		instance.Set(field, value, x, y)
	}
}

func (inter *Interpreter) ScopeIsFunction(scope *Scope) bool {
	if scope.IsFunc {
		return true
	} else if scope.Parent != nil {
		return inter.ScopeIsFunction(scope.Parent)
	}
	return false
}

func (inter *Interpreter) ScopeIsLoop(scope *Scope) bool {
	if scope.IsLoop {
		return true
	} else if scope.Parent != nil {
		return inter.ScopeIsLoop(scope.Parent)
	}
	return false
}

func (inter *Interpreter) SetElementValue(node *SetElem) {
	tableNode, keyNodes := inter.GetTableAndKeys(node.Elem, []Node{})
	if tableNode == nil {
		throw(inter.CurrentFileName, "Attempt to index nothing", node.X, node.Y)
	}

	table := inter.GetNodeValue(tableNode)
	keys := make([]any, len(keyNodes))
	for i, keyNode := range keyNodes {
		key := inter.GetNodeValue(keyNode)
		if cookedValues, ok := key.([]any); ok {
			if len(cookedValues) > 1 {
				throw(inter.CurrentFileName, "Element's key cannot have more than one value.", tableNode.Position(), tableNode.Line())
			} else if len(cookedValues) == 0 {
				throw(inter.CurrentFileName, "Cannot assign an element's key an empty value.", tableNode.Position(), tableNode.Line())
			}

			key = cookedValues[0]
		}

		keys[i] = key
	}

	value := inter.GetNodeValueS(node.Value, node.X, node.Y)

	switch table := table.(type) {
	case *Map:
		inter.SetTableElementValue(table, keys, value, 0, node.X, node.Y)
	default:
		throw(inter.CurrentFileName, "Cannot index non-table value", node.X, node.Y)
	}
}

func (inter *Interpreter) SetFieldValue(node *SetFieldNode) {
	instanceNode, fieldNodes := inter.GetStructAndFieldNames(node.Field, []Node{})
	if instanceNode == nil {
		throw(inter.CurrentFileName, "Attempt to index nothing", node.X, node.Y)
	}

	instance := inter.GetNodeValue(instanceNode)
	fields := make([]string, len(fieldNodes))
	for i, fieldNode := range fieldNodes {
		fields[i] = fieldNode.(*IdentNode).Value
	}

	value := inter.GetNodeValueS(node.Value, node.X, node.Y)

	switch instance := instance.(type) {
	case *StructObject:
		inter.SetInstanceFieldValue(instance, fields, value, 0, node.X, node.Y)
	default:
		throw(inter.CurrentFileName, "Cannot assign field of non-instance value", node.X, node.Y)
	}
}

func (inter *Interpreter) DeclareStructure(structDecl *StructDeclNode) {
	identifier := structDecl.Identifier.Value
	fields := make([]*FieldDecl, len(structDecl.Fields))

	for i, fieldDeclNode := range structDecl.Fields {
		fields[i] = &FieldDecl{
			Identifier: fieldDeclNode.Identifier.Value,
			Method:     fieldDeclNode.Func != nil,
			DataType:   fieldDeclNode.DataType.Value,
			Func:       fieldDeclNode.Func,
		}
	}

	if !inter.CurrentScope.Add(identifier, &Structure{
		Identifier: identifier,
		Fields:     fields,
	}, "struct", structDecl.X, structDecl.Y) {
		throw(inter.CurrentFileName, "Attempt to declare the structure with the same name as the variable '%s'.", structDecl.X, structDecl.Y, identifier)
	}
}

func (inter *Interpreter) NewStructObject(structObjNode *StructNode) *StructObject {
	identifier := structObjNode.Identifier.Value

	originalStructureAny, found := inter.CurrentScope.Get(identifier)
	if !found {
		throw(inter.CurrentFileName, "Attempt to make an instance of structure '%s' that doesn't exist", structObjNode.X, structObjNode.Y, structObjNode.Identifier.Value)
	}

	originalStructure, ok := originalStructureAny.(*Structure)
	if originalStructure == nil || !ok {
		throw(inter.CurrentFileName, "Attempt to make an instance of a nonexistent structure '%s'.", structObjNode.X, structObjNode.Y, identifier)
	}

	structObject := &StructObject{
		Identifier: identifier,
	}

	fields := make([]*Field, len(structObjNode.Fields))
	methods := make([]*Method, originalStructure.CountMethods())

	method_i := 0
	for _, fieldDecl := range originalStructure.Fields {
		if fieldDecl.Func == nil {
			continue
		}
		fieldDecl.Func.Self = structObject

		cell := &Cell{
			Scope: inter.CurrentScope,
		}
		cell.Set(fieldDecl.Func, false, structObjNode.X, structObjNode.Y)

		methods[method_i] = &Method{
			Identifier: fieldDecl.Identifier,
			Func:       cell,
		}
		method_i++
	}

	for i, fieldNode := range structObjNode.Fields {
		fieldName := fieldNode.Identifier.Value
		if !originalStructure.CheckField(fieldName) {
			throw(inter.CurrentFileName, "Attempt to assign a nonexistent field '%s' of structure '%s' while trying to make an instance.", structObjNode.X, structObjNode.Y, fieldName, identifier)
		}
		if originalStructure.IsAFunc(fieldName) {
			throw(inter.CurrentFileName, "Attempt to assign a value for a method '%s' of structure '%s'.", structObjNode.X, structObjNode.Y, fieldName, identifier)
		}

		v := inter.GetNodeValueS(fieldNode.Value, fieldNode.Identifier.X, fieldNode.Identifier.Y)

		cell := &Cell{
			Scope: inter.CurrentScope,
		}

		originalStructField := originalStructure.GetField(fieldName)

		switch v := v.(type) {
		case []any:
			if len(v) > 1 {
				throw(inter.CurrentFileName, "Field cannot have more than one value.", fieldNode.Identifier.X, fieldNode.Identifier.Y)
			} else if len(v) == 0 {
				throw(inter.CurrentFileName, "Cannot assign a field an empty value.", fieldNode.Identifier.X, fieldNode.Identifier.Y)
			}

			cell.InitFromRaw(v[0], originalStructField.DataType, false, structObjNode.X, structObjNode.Y)
		default:
			cell.InitFromRaw(v, originalStructField.DataType, false, structObjNode.X, structObjNode.Y)
		}

		fields[i] = &Field{
			Identifier: fieldNode.Identifier.Value,
			DataType:   cell.DataType,
			Value:      cell,
		}
	}

	structObject.Fields = fields
	structObject.Methods = methods
	structObject.ToMemoryLayout(structObject.Layout())

	return structObject
}

func (inter *Interpreter) CookValues(max_i uint, values [][]Node, x, y int) []any {
	readyValues := make([]any, 0, len(values))

	for i := uint(0); i < max_i; i++ {
		if i >= uint(len(values)) {
			break
		}
		node := values[i]

		value := inter.GetNodeValueS(node, x, y)

		switch value := value.(type) {
		case []any:
			readyValues = append(readyValues, value...)
		default:
			readyValues = append(readyValues, value)
		}
	}

	if uint(len(readyValues)) < max_i {
		readyValues = append(readyValues, make([]any, max_i-uint(len(readyValues)))...)
	}

	return readyValues
}

type ReturnNil struct{}

func (inter *Interpreter) CompleteNode(node Node) (end, skip bool, value []any) {
	scope := inter.CurrentScope
	switch node := node.(type) {
	case *FuncDec:
		if len(node.Identifier.Value) == 0 {
			throw(inter.CurrentFileName, "Name of the function cannot be empty.", node.X, node.Y)
		}
		if !inter.CurrentScope.Add(node.Identifier.Value, node, "func", node.X, node.Y) {
			throw(inter.CurrentFileName, "Attempt to redeclare a variable '%s'.", node.X, node.Y, node.Identifier.Value)
		}
	case *StructDeclNode:
		inter.DeclareStructure(node)
	case *VarDec:
		readyValues := inter.CookValues(uint(len(node.Identifier)), node.Value, node.X, node.Y)

		if len(readyValues) > len(node.Identifier) && !node.Argument {
			throw(inter.CurrentFileName, "Too many values(%d) for %d identifier(s).", node.X, node.Y, len(readyValues), len(node.Identifier))
		} else if len(readyValues) > len(node.Identifier) && node.Argument {
			throw(inter.CurrentFileName, "Attempt to use multiple values as a single argument.", node.X, node.Y)
		}

		for i, ident := range node.Identifier {
			if ident.Value == "_" {
				continue
			}

			if !inter.CurrentScope.Add(ident.Value, readyValues[i], node.DataTypes[i].Value, node.X, node.Y) {
				throw(inter.CurrentFileName, "Attempt to redeclare a variable '%s'.", node.X, node.Y, ident.Value)
			}
		}
	case *SetVar:
		readyValues := inter.CookValues(uint(len(node.Value)), node.Value, node.X, node.Y)

		if len(readyValues) > len(node.Var) {
			throw(inter.CurrentFileName, "Too many values in assignment", node.X, node.Y)
		} else if len(readyValues) < len(node.Var) {
			throw(inter.CurrentFileName, "Too few values in assignment", node.X, node.Y)
		}

		for i, ident := range node.Var {
			if !inter.CurrentScope.Set(ident.Value, readyValues[i], node.X, node.Y) {
				throw(inter.CurrentFileName, "Attempt to assign value to non-existing variable '%s'.", node.X, node.Y, node.Value)
			}
		}
	case *IndirAssignNode:
		valuePointerInterface := inter.GetNodeValue(node.Pointer)

		valuePointer, ok := valuePointerInterface.(uintptr)
		if !ok {
			throw(inter.CurrentFileName, "Attempt to do indirect assignment with invalid pointer.", node.X, node.Y)
		}

		pointerCell := scope.GetCellWithAddress(unsafe.Pointer(valuePointer))
		if pointerCell == nil {
			throw(inter.CurrentFileName, "Attempt to do indirect assignment of non-existing pointer.", node.X, node.Y)
		}

		valuePtr, ok := pointerCell.Get().(uintptr)
		if !ok {
			throw(inter.CurrentFileName, "Attempt to do indirect assignment with non-pointer value.", node.X, node.Y)
		}

		cellOfPtr := scope.GetCellWithAddress(unsafe.Pointer(valuePtr))
		if cellOfPtr == nil {
			throw(inter.CurrentFileName, "Attempt to do indirect assignment of non-existing value.", node.X, node.Y)
		}

		newValue := inter.GetNodeValueS(node.Value, node.X, node.Y)

		cellOfPtr.Set(newValue, false, node.X, node.Y)
	case *FuncCall:
		inter.GetNodeValue(node)
	case *SetElem:
		inter.SetElementValue(node)
	case *IfStmt:
		result := inter.GetNodeValueS(node.Condition, node.X, node.Y)
		if result == true {
			return inter.CompleteBody(node.Body, false, false)
		} else if result == false && node.Else != nil {
			return inter.CompleteNode(node.Else)
		}
	case *ElseStmt:
		if len(node.Condition) > 0 {
			result := inter.GetNodeValueS(node.Condition, node.X, node.Y)
			if result == true {
				return inter.CompleteBody(node.Body, false, false)
			} else if result == false && node.Else != nil {
				return inter.CompleteNode(node.Else)
			}
		} else {
			return inter.CompleteBody(node.Body, false, false)
		}
	case *ContinueNode:
		return false, true, nil
	case *BreakNode:
		return true, false, nil
	case *ReturnNode:
		readyValues := inter.CookValues(uint(len(node.Value)), node.Value, node.X, node.Y)

		return true, false, readyValues
	case *ExternalImport:
		if inter.UnableToImport {
			throw(inter.CurrentFileName, "External import keyword must be at the beggining of the code.", node.Position(), node.Line())
		}
		scope := inter.CurrentScope
		if !scope.MainScope {
			throw(inter.CurrentFileName, "Cannot use external import keyword outside main scope.", node.Position(), node.Line())
		}

		path := node.Path.Value
		if len(path) > 0 {
			path := node.Path.Value

			loadLibraryIntoScope(inter.CurrentFileName, path, node, scope)

		} else {
			throw(inter.CurrentFileName, "Cannot perform external import without a library path.", node.Position(), node.Line())
		}
		return false, false, nil
	case *Import:
		if inter.UnableToImport {
			throw(inter.CurrentFileName, "Import keyword must be at the beggining of the code.", node.Position(), node.Line())
		}
		if !inter.CurrentScope.MainScope {
			throw(inter.CurrentFileName, "Cannot use import keyword outside main scope.", node.Position(), node.Line())
		}

		if len(node.Path) > 0 && len(node.Path) < 2 {
			path, ok := node.Path[0].(*StrNode)
			if !ok {
				throw(inter.CurrentFileName, "Path for the import keyword cannot be a non-string value.", node.Position(), node.Line())
			}

			importModule(path.Value, inter.CurrentScope, node.X, node.Y)
		} else if len(node.Path) > 1 {
			throw(inter.CurrentFileName, "Cannot import more than one file or module.", node.Position(), node.Line())
		} else {
			throw(inter.CurrentFileName, "Cannot import the file or the module without a path.", node.Position(), node.Line())
		}
		return false, false, nil
	case *WhileNode:
		for cond := inter.GetNodeValueS(node.Condition, node.X, node.Y); cond == true; cond = inter.GetNodeValueS(node.Condition, node.X, node.Y) {
			end, skip, value := inter.CompleteBody(node.Body, false, true)
			if skip {
				continue
			} else if end || value != nil {
				return end, skip, value
			}
		}
	case *ForeachNode:
		cycleValue := inter.GetNodeValueS(node.CycleValue, node.X, node.Y)
		keyIdent, valueIdent := node.KeyIdent, node.ValueIdent

		switch cycleValue := cycleValue.(type) {
		case *Map:
			for key, value := range cycleValue.AllFromFront() {
				end, skip, returnValue := inter.CompleteBody(node.Body, false, true, [3]any{keyIdent.Value, key, getValueType(key)}, [3]any{valueIdent.Value, value.Get(), value.DataType})

				if skip {
					continue
				} else if end || returnValue != nil {
					return end, skip, returnValue
				}
			}
		default:
			throw(inter.CurrentFileName, "Unable to iterate over a non-table value.", node.X, node.Y)
		}
	default:
		//fmt.Printf("%T",node.(*BinOpNode).L)
		throw(inter.CurrentFileName, "Invalid node '%s'.", node.Position(), node.Line(), getInterfaceType(node))
	}
	inter.UnableToImport = true
	return false, false, nil
}

func (inter *Interpreter) Complete(logenv bool) map[any]*Cell { //go run yks run test.yks
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		for externalTaskArgs := range externalCallingChan {
			taskInter := externalTaskArgs.Inter

			r1, r2, err := syscallAddress(taskInter, externalTaskArgs.FuncCall, uint(len(externalTaskArgs.FuncCall.Arguments)), externalTaskArgs.ArgsValues, externalTaskArgs.Addr)

			externalCallFinished <- ExternalTaskResult{
				r1, r2, err,
			}
		}
	}()

	mainScope := NewScope(inter, nil)
	mainScope.MainScope = true

	inter.CurrentScope = mainScope

	for ident, function := range builtinFuncs {
		mainScope.Add(ident, newFTemp(ident, function), "func", -2, -2)
	}

	for _, node := range inter.AST {
		inter.CompleteNode(node)
	}

	if logenv {
		fmt.Println(mainScope.Data)
	}
	clear(mainScope.Pointers)

	return mainScope.Data
}
