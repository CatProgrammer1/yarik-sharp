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

type Scope struct {
	Data           map[any]*Cell
	Pointers       map[unsafe.Pointer]*Cell
	Parent         *Scope
	IsFunc, IsLoop bool
	ImportedLibs   []string
	MainScope      bool
}

type Cell struct {
	IntValue      int64
	FloatValue    float64
	BoolValue     bool
	StringValue   string
	StructValue   *Structure
	InstanceValue *StructObject
	TableValue    *Map
	FuncValue     *FuncDec
	PtrValue      uintptr
	ErrorValue    error

	DataType string // int, float, string, bool, struct, instance, table, ptr, func, error, "valueptr"
	Ptr      unsafe.Pointer
	TempBuf  any
	Pinner   *runtime.Pinner

	Scope *Scope
}

func (cell *Cell) Set(value any) {
	if cell.Pinner == nil {
		cell.Pinner = new(runtime.Pinner)
	} else {
		cell.Pinner.Unpin()
	}

	switch value := value.(type) {
	case int64:
		cell.IntValue = value
		cell.DataType = "int"
		cell.Ptr = unsafe.Pointer(&cell.IntValue)
	case float64:
		cell.FloatValue = value
		cell.DataType = "float"
		cell.Ptr = unsafe.Pointer(&cell.FloatValue)
	case bool:
		cell.BoolValue = value
		cell.DataType = "bool"
		cell.Ptr = unsafe.Pointer(&cell.BoolValue)
	case string:
		cell.StringValue = value
		cell.DataType = "string"

		ptr, buf := valueToPtr(cell.StringValue, 0, 0)
		cell.TempBuf = buf

		cell.Ptr = unsafe.Pointer(ptr)
	case *StructObject:
		cell.InstanceValue = value
		cell.DataType = "instance"
		cell.Ptr = unsafe.Pointer(value.Address())
	case *Structure:
		cell.StructValue = value
		cell.DataType = "struct"
		cell.Ptr = unsafe.Pointer(value)
	case *FuncDec:
		cell.FuncValue = value
		cell.DataType = "func"
		cell.Ptr = unsafe.Pointer(value)
	case *Map:
		cell.TableValue = value
		cell.DataType = "table"

		value.ToMemory()

		cell.Ptr = unsafe.Pointer(value.Address())
	case unsafe.Pointer:
		cell.PtrValue = uintptr(value)
		cell.DataType = "ptr"
		cell.Ptr = unsafe.Pointer(&cell.PtrValue)
	case uintptr:
		cell.PtrValue = value
		cell.DataType = "ptr"
		cell.Ptr = unsafe.Pointer(&cell.PtrValue)
	case error:
		cell.ErrorValue = value
		cell.DataType = "error"
		cell.Ptr = unsafe.Pointer(&cell.ErrorValue)
	default:
		fmt.Printf("%T\n", value)
		panic("Unsupported type in Cell.Set")
	}
	cell.Pinner.Pin(cell.Ptr)
}

func (cell *Cell) Get() any {
	switch cell.DataType {
	case "int":
		return cell.IntValue
	case "float":
		return cell.FloatValue
	case "bool":
		return cell.BoolValue
	case "string":
		return cell.StringValue
	case "instance":
		return cell.InstanceValue
	case "struct":
		return cell.StructValue
	case "table":
		return cell.TableValue
	case "ptr":
		return cell.PtrValue
	case "func":
		return cell.FuncValue
	case "error":
		return cell.ErrorValue
	default:
		fmt.Println(cell.DataType)
		panic("Unsupported type in Cell.Get")
	}
}

func (cell *Cell) GetAddress() unsafe.Pointer {
	return cell.Ptr
}

func CLPTR(v any) *Cell {
	cell := &Cell{}

	cell.Set(v)

	return cell
}

func CL(v any) Cell {
	cell := Cell{}

	cell.Set(v)

	return cell
}

func checkType[T any](v any) bool {
	_, ok := v.(T)
	return ok
}

type Map struct {
	*orderedmap.OrderedMap[Cell, *Cell]

	Bits     int8
	Pointers []any
	Layout   []string
	Mem      []byte
}

func anyToBytes(v []any, m *Map) []byte {
	buf := new(bytes.Buffer)
	fmt.Println("SIGMA TRIGGER", v)

	m.Layout = []string{}
	m.Pointers = []any{}

	for _, x := range v {
		switch t := x.(type) {
		case int64:
			layout := "int"
			if m.Bits != 0 {
				layout += numtostr(int64(m.Bits))

				m.Layout = append(m.Layout, layout)
				m.Pointers = append(m.Pointers, nil)

				binary.Write(buf, binary.LittleEndian, toInt(t, int(m.Bits)))
				break
			}

			m.Layout = append(m.Layout, layout)
			m.Pointers = append(m.Pointers, nil)

			binary.Write(buf, binary.LittleEndian, t)
		case float64:
			layout := "float"
			if m.Bits != 0 {
				layout = "int" + numtostr(m.Bits)

				m.Layout = append(m.Layout, layout)
				m.Pointers = append(m.Pointers, nil)

				binary.Write(buf, binary.LittleEndian, ftoInt(t, int(m.Bits)))
				break
			}

			m.Layout = append(m.Layout, layout)
			m.Pointers = append(m.Pointers, nil)

			binary.Write(buf, binary.LittleEndian, t)
		case bool:
			m.Layout = append(m.Layout, "bool")
			m.Pointers = append(m.Pointers, nil)

			binary.Write(buf, binary.LittleEndian, t)
		case string:
			m.Layout = append(m.Layout, "string")
			m.Pointers = append(m.Pointers, nil)

			binary.Write(buf, binary.LittleEndian, append([]byte(t), 0))
		case error:
			m.Layout = append(m.Layout, "error")
			m.Pointers = append(m.Pointers, nil)

			binary.Write(buf, binary.LittleEndian, append([]byte(t.Error()), 0))
		case uintptr:
			m.Layout = append(m.Layout, "ptr")
			m.Pointers = append(m.Pointers, nil)

			binary.Write(buf, binary.LittleEndian, uint64(t))
		case unsafe.Pointer:
			m.Layout = append(m.Layout, "ptr")
			m.Pointers = append(m.Pointers, nil)

			binary.Write(buf, binary.LittleEndian, uint64(uintptr(t)))
		case *Map:
			m.Layout = append(m.Layout, "table")
			m.Pointers = append(m.Pointers, t)

			binary.Write(buf, binary.LittleEndian, uint32(len(t.Mem)))
			buf.Write(t.Mem)
		default:
			panic("Unsupported type")
		}
	}
	return buf.Bytes()
}

func bytesToAny(mem []byte, layout []string, pointers []any) []any {
	var res []any
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
			res = append(res, m)
		case "int", "int64":
			var v int64
			binary.Read(r, binary.LittleEndian, &v)

			res = append(res, v)
		case "int32":
			var v int32
			binary.Read(r, binary.LittleEndian, &v)

			res = append(res, int64(v))
		case "int16":
			var v int16
			binary.Read(r, binary.LittleEndian, &v)

			res = append(res, int64(v))
		case "int8":
			var v int8
			binary.Read(r, binary.LittleEndian, &v)

			res = append(res, int64(v))
		case "ptr", "uint64":
			var v uint64
			binary.Read(r, binary.LittleEndian, &v)

			res = append(res, uintptr(v))
		case "float":
			var v float64
			binary.Read(r, binary.LittleEndian, &v)

			res = append(res, v)
		case "bool":
			var b byte
			binary.Read(r, binary.LittleEndian, &b)

			res = append(res, b != 0)
		case "string":
			var ln uint32
			handle(binary.Read(r, binary.LittleEndian, &ln))

			b := make([]byte, ln)
			_, err := r.Read(b)
			handle(err)

			res = append(res, string(b))
		case "error":
			var ln uint32
			handle(binary.Read(r, binary.LittleEndian, &ln))

			b := make([]byte, ln)
			_, err := r.Read(b)
			handle(err)

			res = append(res, errors.New(string(b)))
		default:
			panic("Unsupported type: " + t)
		}
	}

	return res
}

func (m *Map) ToMemory() {
	m.Mem = anyToBytes(mapToSliceAny(m), m)
}

func (m *Map) FromMemory() {
	s := bytesToAny(m.Mem, m.Layout, m.Pointers)

	i := 0
	for _, value := range m.AllFromFront() {
		value.Set(s[i])
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

func format(v ...any) string {
	var formated string

	for i, a := range v {
		var suffix string
		if i != len(v)-1 {
			suffix = " "
		}

		switch a := a.(type) {
		case *Map:
			mapFormat := "{%s}"
			elemFormat := "[%s]: %s,"

			var elements string
			i := 0
			for k, v := range a.AllFromFront() {
				var elementSuffix string
				if i != a.Len()-1 {
					elementSuffix = " "
				}

				elements += fmt.Sprintf(elemFormat+elementSuffix, format(k.Get()), format(v.Get()))
				i++
			}

			formated += fmt.Sprintf(mapFormat, elements) + suffix
		case error:
			formated += fmt.Sprint(a.Error()) + suffix
		case *FuncDec, *Structure:
			formated += fmt.Sprintf("%p", a) + suffix
		case *StructObject:
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

func importModule(path string, mainScope *Scope) {
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
			cell := &Cell{}
			cell.Set(v.Get())

			mainScope.Data[k] = cell
			mainScope.Pointers[cell.Ptr] = cell
		}
		return
	}
	throwNoPos("Invalid file or library '%s'", path)
}

func mapToSlice[T any](m *Map) []T {
	slice := make([]T, m.Len())

	var t T

	for k, v := range m.AllFromFront() {
		kind := reflect.ValueOf(t).Kind()

		switch kind {
		case reflect.Uint8:
			if reflect.ValueOf(v.Get()).Kind() == reflect.Float64 {
				fk, _ := numberToFloat64(k)
				slice[int(fk)] = any(byte(v.Get().(float64))).(T)
				break
			}
			fallthrough
		default:
			fk, _ := numberToFloat64(k)
			slice[int(fk)] = v.Get().(T)
		}
	}

	return slice
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

func NewScope(parent *Scope) *Scope {
	return &Scope{
		Data:     make(map[any]*Cell),
		Pointers: make(map[unsafe.Pointer]*Cell),
		Parent:   parent,
	}
}

func ptrToAny(v any) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&v))
}

func (scope *Scope) Add(key, value any) (success bool) {
	if key == "_" {
		return true
	}
	if _, ok := scope.Data[key]; ok {
		return false
	}
	cell := &Cell{}
	cell.Set(value)

	scope.Data[key] = cell
	scope.Pointers[cell.Ptr] = cell
	switch value := value.(type) {
	case *StructObject:
		for _, field := range value.Fields {
			fcell := field.Value

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
			throw("Assignment to non-variable value", x, y)
		}

		cell := scope.Data[key]
		if cell.Ptr != nil {
			delete(scope.Pointers, cell.Ptr)
		}
		cell.Set(value)

		return true
	} else if scope.Parent != nil {
		return scope.Parent.Set(key, value, x, y)
	}
	return false
}

func (scope *Scope) Get(key any) any {
	v, ok := scope.Data[key]
	if ok {
		return v.Get()
	} else if scope.Parent != nil {
		return scope.Parent.Get(key)
	}
	return nil
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

func (structure *Structure) IsAFunc(name string) bool {
	for _, field := range structure.Fields {
		if field.Identifier == name {
			return field.Func != nil
		}
	}
	return false
}

type FieldDecl struct {
	Identifier string
	Method     bool
	Bits       int8
	Func       *FuncDec
}

type StructObject struct {
	Identifier string
	Fields     []*Field
	LastMem    []byte
}

type FieldLayout struct {
	Name   string
	Offset uintptr
	Size   uintptr
	Type   string // например "uint32", "uintptr"
}

func (s *StructObject) ToMemoryLayout(layout []FieldLayout) []byte {
	size := layout[len(layout)-1].Offset + layout[len(layout)-1].Size

	var mem []byte
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
		case "int8":
			v := int8(toInt64(val.Get()))
			mem[offset] = byte(v)

		case "uint8":
			v := uint8(toInt64(val.Get()))
			mem[offset] = byte(v)

		case "int16":
			v := int16(toInt64(val.Get()))
			binary.LittleEndian.PutUint16(mem[offset:], uint16(v))

		case "uint16":
			v := uint16(toInt64(val.Get()))
			binary.LittleEndian.PutUint16(mem[offset:], v)

		case "int32":
			v := int32(toInt64(val.Get()))
			binary.LittleEndian.PutUint32(mem[offset:], uint32(v))

		case "uint32":
			v := uint32(toInt64(val.Get()))
			binary.LittleEndian.PutUint32(mem[offset:], v)

		case "int64":
			v := int64(toInt64(val.Get()))
			binary.LittleEndian.PutUint64(mem[offset:], toUint64(v))

		case "uint64", "uintptr", "ptr":
			v := toUint64(val.Get())
			binary.LittleEndian.PutUint64(mem[offset:], v)

		case "float":
			binary.LittleEndian.PutUint64(mem[offset:], toUint64(val.Get()))
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

func (s *StructObject) FromMemoryLayout(layout []FieldLayout) {
	if s.LastMem == nil {
		return
	}
	mem := s.LastMem

	for _, lf := range layout {
		offset := int(lf.Offset)

		switch lf.Type {
		case "int8":
			v := int8(mem[offset])
			s.Set(lf.Name, int64(v))

		case "uint8":
			v := mem[offset]
			s.Set(lf.Name, int64(v))

		case "int16":
			v := int16(binary.LittleEndian.Uint16(mem[offset:]))
			s.Set(lf.Name, int64(v))

		case "uint16":
			v := binary.LittleEndian.Uint16(mem[offset:])
			s.Set(lf.Name, int64(v))

		case "int32":
			v := int32(binary.LittleEndian.Uint32(mem[offset:]))
			s.Set(lf.Name, int64(v))

		case "uint32":
			v := binary.LittleEndian.Uint32(mem[offset:])
			s.Set(lf.Name, int64(v))

		case "int64":
			v := int64(binary.LittleEndian.Uint64(mem[offset:]))
			s.Set(lf.Name, v)

		case "uint64", "uintptr", "ptr":
			v := binary.LittleEndian.Uint64(mem[offset:])
			s.Set(lf.Name, uintptr(v))

		case "float":
			bits := binary.LittleEndian.Uint64(mem[offset:])
			v := math.Float64frombits(bits)
			s.Set(lf.Name, v)
		case "bool":
			v := mem[offset]
			s.Set(lf.Name, v == 1)
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

			// Берём данные из памяти по offset
			if offset+int(subSize) > len(mem) {
				panic("memory slice out of bounds")
			}
			subMem := make([]byte, subSize)
			copy(subMem, mem[offset:offset+int(subSize)])

			sub.LastMem = subMem
			sub.FromMemoryLayout(subLayout)
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
		return int8(v)
	case 16:
		return int16(v)
	case 32:
		return int32(v)
	case 64:
		return int64(v)
	case 0:
		return v
	default:
		panic("invalid bit size")
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
	case 0:
		return v
	default:
		panic("invalid bit size")
	}
}

func toInt64(v any) int64 {
	switch val := v.(type) {
	case int:
		return int64(val)
	case int32:
		return int64(val)
	case uint32:
		return int64(val)
	case int64:
		return int64(val)
	case uint64:
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
	case int:
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
	layout := make([]FieldLayout, 0, len(s.Fields))
	offset := uintptr(0)

	for _, field := range s.Fields {
		var size, align uintptr
		var typ string
		cell := field.Value

		if field.LayoutType != 0 {
			prefix := ""
			if field.LayoutType > 0 {
				prefix = "u"
			}
			switch int64(math.Abs(float64(field.LayoutType))) {
			case 8:
				size, align, typ = 1, 1, prefix+"int8"
			case 16:
				size, align, typ = 2, 2, prefix+"int16"
			case 32:
				size, align, typ = 4, 4, prefix+"int32"
			case 64:
				size, align, typ = 8, 8, prefix+"int64"
			default:
				throwNoPos("Unsupported amount of bits: %d", field.LayoutType)
			}
		} else {
			switch v := cell.Get().(type) {
			case *StructObject:
				sub := v.Layout()
				last := sub[len(sub)-1]
				size = last.Offset + last.Size

				align, typ = 8, "instance"
			case unsafe.Pointer, uintptr:
				size, align, typ = 8, 8, "ptr"
			case int64:
				size, align, typ = 8, 8, "int64"
			case bool:
				size, align, typ = 1, 1, "bool"
			case float64:
				size, align, typ = 8, 8, "float"
			default:
				throwNoPos("Unsupported field type: %s", field.Identifier)
			}
		}

		offset = alignf(offset, align)

		layout = append(layout, FieldLayout{
			Name:   field.Identifier,
			Offset: offset,
			Size:   size,
			Type:   typ,
		})

		offset += size
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
	return nil, false
}

func (structObj *StructObject) GetCell(fieldName string) (*Cell, bool) {
	for _, field := range structObj.Fields {
		if field.Identifier == fieldName {
			cell := field.Value
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

func (structObj *StructObject) Set(fieldName string, value any) bool {
	for _, field := range structObj.Fields {
		cell := field.Value
		if field.Method {
			method := cell.Get().(*FuncDec)

			throw("Cannot assign value to a instance's method.", method.X, method.Y)
		}
		if field.Identifier == fieldName {
			cell.Set(value)

			return true
		}
	}
	return false
}

type Field struct {
	Identifier string
	Method     bool
	LayoutType int8
	Value      *Cell
}

func newInstance(name string, fields []*Field) *StructObject {
	return &StructObject{
		Identifier: name,
		Fields:     fields,
		LastMem:    []byte{},
	}
}

type Interpreter struct {
	AST            []Node
	CurrentScope   *Scope
	UnableToImport bool
}

func NewInterpreter(ast []Node) *Interpreter {
	return &Interpreter{
		AST: ast,
	}
}

func (inter *Interpreter) GetBinOpValue(node *BinOpNode) any {
	if node.operator == "sub" && node.L == nil && node.R != nil {
		value := inter.GetNodeValue(node.R)

		rtype := checkDataType("number", value)
		if rtype {
			return -mustNTOF64(value)
		}
		throw("Unable to use unary operator '-' on non-number value.", node.X, node.Y)
	}

	err := "Cannot perform binary operations on multiple values at the same time."

	l, r := inter.GetNodeValue(node.L), inter.GetNodeValue(node.R)

	returnL, ok := l.([]any)
	if ok {
		if len(returnL) > 1 {
			throw(err, node.X, node.Y)
		}
		l = returnL[0]
	}

	returnR, ok := r.([]any)
	if ok {
		if len(returnR) > 1 {
			throw(err, node.X, node.Y)
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

	return f(l, r, node.X, node.Y)
}

func (inter *Interpreter) GetNodeValue(node Node) any {
	switch node := node.(type) {
	case *NilNode:
		return nil
	case *IntNode:
		return node.Value
	case *FloatNode:
		return node.Value
	case *StrNode:
		return node.Value
	case *BoolNode:
		return node.Value
	case *BinOpNode:
		return inter.GetBinOpValue(node)
	case *Brackets:
		return inter.GetNodeValueS(node.Value, node.X, node.Y)
	case *MapNode:
		return inter.GetMap(node)
	case *FuncDec:
		return node
	case *FuncCall:
		return inter.CallFunction(node)
	case *IdentNode:
		v := inter.CurrentScope.Get(node.Value)

		return v
	case *StructNode:
		return inter.NewStructObject(node)
	case *GetPtrNode:
		if node.Src == nil {
			throw("Attempt to get a pointer of nothing.", node.X, node.Y)
		}
		srcNode := node.Src

		scope := inter.CurrentScope

		switch srcNode := srcNode.(type) {
		case *IdentNode:
			identifier := srcNode.Value

			cell := scope.GetCell(identifier)
			if cell == nil {
				throw("Attempt to get a pointer of non-existing value.", node.X, node.Y)
			}
			if cell.Ptr == nil {
				throw("Attempt to get pointer of nil value.", node.X, node.Y)
			}

			return uintptr(cell.Ptr)
		case *GetElementNode:
			tableNode, keyNodes := inter.GetTableAndKeys(srcNode, []Node{})
			if tableNode == nil {
				throw("Attempt to index nothing.", srcNode.X, srcNode.Y)
			}

			table := inter.GetNodeValue(tableNode)

			keys := []any{}
			for _, keyNode := range keyNodes {
				keys = append(keys, inter.GetNodeValue(keyNode))
			}

			switch table := table.(type) {
			case *Map, string:
				cell := inter.GetTableCellByKeys(table, keys, srcNode, 0)

				return cell.Ptr
			default:
				throw("Cannot index non-table value.", node.X, node.Y)
			}
		case *GetFieldNode:
			cell := inter.GetInstanceFieldCell(srcNode)

			return cell.Ptr
		}
	case *GetFieldNode:
		cell := inter.GetInstanceFieldCell(node)

		/*structObjNode, fieldNodes := inter.GetStructAndFieldNames(node, []Node{})
		if structObjNode == nil {
			throw("Attempt to get field of nothing.", node.X, node.Y)
		}

		structObj, ok := inter.GetNodeValue(structObjNode).(*StructObject)
		if !ok {
			throw("Attempt to get field of a non-structure value.", structObjNode.Position(), structObjNode.Line())
		}

		fields := []string{}
		for _, fieldNode := range fieldNodes {

			fieldIdentNode, ok := fieldNode.(*IdentNode)
			if !ok {
				throw("Field name must be an identifier", fieldNode.Position(), fieldNode.Line())
			}

			fields = append(fields, fieldIdentNode.Value)
		}

		return inter.GetFieldValueByNames(structObj, fields, node, 0)*/

		return cell.Get()
	case *GetElementNode:
		tableNode, keyNodes := inter.GetTableAndKeys(node, []Node{})
		if tableNode == nil {
			throw("Attempt to index nothing.", node.X, node.Y)
		}

		table := inter.GetNodeValue(tableNode)

		keys := []any{}
		for _, keyNode := range keyNodes {
			keys = append(keys, inter.GetNodeValue(keyNode))
		}

		switch table := table.(type) {
		case *Map, string:
			return inter.GetTableValueByKeys(table, keys, node, 0)
		default:
			throw("Cannot index non-table or non-string value.", node.X, node.Y)
		}
	}
	throw("Invalid node '%s'.", node.Position(), node.Line(), getInterfaceType(node))
	return nil
}

func (inter *Interpreter) GetNodeValueS(nodes []Node, x, y int) any {
	if len(nodes) > 1 || len(nodes) == 0 {
		throw("Value has more than one value or is empty", x, y)
	}
	return inter.GetNodeValue(nodes[0])
}

func (inter *Interpreter) GetTableAndKeys(node *GetElementNode, keys []Node) (Node, []Node) {
	if len(node.Map) > 1 {
		throw("Cannot index more than one value at the same time", node.X, node.Y)
	}
	if len(node.Key) > 1 {
		throw("Key cannot have more than one value", node.X, node.Y)
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

	tableCell, iscell := table.(*Cell)
	if iscell {
		table = tableCell.Get()
	}

	switch table := table.(type) {
	case *Map:
		elem := table.GetElement(CL(key)).Value
		var val any
		if elem != nil {
			val = elem.Get()
		}

		if index+1 < len(keys) {
			return inter.GetTableValueByKeys(val, keys, getElemN, index+1)
		}
		return val
	case string:
		key, ok := numberToFloat64(key)
		if ok {
			i := int(key)
			if i >= len(table) || i < 0 {
				throw("Attempt to index a character beyond the string limit.", getElemN.X, getElemN.Y)
			}
			if index+1 != len(keys) {
				throw("Repeated indexing of a character is not allowed.", getElemN.X, getElemN.Y)
			}

			char := string([]rune(table)[i])

			return char
		}
	}
	throw("Attempt to index non-table value.", getElemN.X, getElemN.Y)
	return nil
}

func (inter *Interpreter) GetTableCellByKeys(table any, keys []any, getElemN *GetElementNode, index int) *Cell {
	if index >= len(keys) {
		return nil
	}

	key := keys[index]

	switch table := table.(type) {
	case *Map:
		elem := table.GetElement(CL(key)).Value
		var val *Cell
		if elem != nil {
			val = elem
		}

		if index+1 < len(keys) {
			return inter.GetTableCellByKeys(val.Get(), keys, getElemN, index+1)
		}
		return val
	}
	throw("Attempt to index non-table value.", getElemN.X, getElemN.Y)
	return nil
}

func (inter *Interpreter) GetStructAndFieldNames(node *GetFieldNode, fields []Node) (Node, []Node) {
	if len(node.Field) > 1 {
		throw("Cannot get value of more than one field at the same time.", node.X, node.Y)
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
		throw("Attempt to get a value of nonexistent field '%s'", getFieldN.X, getFieldN.Y, fieldName)
	}

	if index+1 < len(fieldNames) {
		nextStructObj, ok := val.(*StructObject)
		if !ok {
			throw("Attempt to get field of a non-structure value", getFieldN.X, getFieldN.Y)
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
		throw("Attempt to get a value of nonexistent field '%s'", getFieldN.X, getFieldN.Y, fieldName)
	}

	if index+1 < len(fieldNames) {
		nextStructObj, ok := val.Get().(*StructObject)
		if !ok {
			throw("Attempt to get field of a non-structure value", getFieldN.X, getFieldN.Y)
		}

		return inter.GetFieldCellByNames(nextStructObj, fieldNames, getFieldN, index+1)
	}
	return val
}

func (inter *Interpreter) GetInstanceFieldCell(getFieldNode *GetFieldNode) *Cell {
	structObjNode, fieldNodes := inter.GetStructAndFieldNames(getFieldNode, []Node{})
	if structObjNode == nil {
		throw("Attempt to get field of nothing.", getFieldNode.X, getFieldNode.Y)
	}

	structObj, ok := inter.GetNodeValue(structObjNode).(*StructObject)
	if !ok {
		throw("Attempt to get field of a non-structure value.", structObjNode.Position(), structObjNode.Line())
	}

	fields := []string{}
	for _, fieldNode := range fieldNodes {

		fieldIdentNode, ok := fieldNode.(*IdentNode)
		if !ok {
			throw("Field name must be an identifier", fieldNode.Position(), fieldNode.Line())
		}

		fields = append(fields, fieldIdentNode.Value)
	}

	return inter.GetFieldCellByNames(structObj, fields, getFieldNode, 0)
}

func (inter *Interpreter) GetMap(node *MapNode) *Map {
	m := orderedmap.NewOrderedMap[Cell, *Cell]()

	for _, element := range node.Map {
		key, value := inter.GetNodeValueS(element.Key, element.X, element.Y), inter.GetNodeValueS(element.Value, element.X, element.Y)

		values, ok := value.([]any)
		if ok {
			if len(values) > 1 {
				throw("Field cannot have more than one value.", element.X, element.Y)
			} else if len(values) == 0 {
				throw("Cannot assign a field cannot be an empty value.", element.X, element.Y)
			}

			m.Set(CL(key), CLPTR(values[0]))
		} else {
			m.Set(CL(key), CLPTR(value))
		}
	}

	b := int8(0)
	if node.Bits != nil {
		b = int8(node.Bits.Value)
	}

	fmap := &Map{m, b, []any{}, []string{}, []byte{}}
	fmap.ToMemory()

	return fmap
}

func (inter *Interpreter) Current(scope *Scope) {
	inter.CurrentScope = scope
}

func (inter *Interpreter) CallFunction(node *FuncCall) []any {
	funcDec, ok := inter.GetNodeValue(node.Func).(*FuncDec)
	if !ok {
		throw("Attempt to call a non-function object.", node.X, node.Y)
	}

	if funcDec.Template != nil {
		args := []any{node.X, node.Y, inter}

		argsValues := [][]Node{}
		for _, argNode := range node.Arguments {
			argsValues = append(argsValues, []Node{argNode})
		}

		return append([]any{}, funcDec.Template(
			append(args, inter.CookValues(len(node.Arguments), argsValues, node.X, node.Y)...)...,
		)...,
		)
	}

	body := funcDec.Body
	argsBody := []Node{}

	if len(node.Arguments) > len(funcDec.Arguments) {
		throw("Attempt to pass more arguments to a function call than function actually need.", node.X, node.Y)
	}

	argsIdentifiers := funcDec.Arguments //[]IdentNode{}
	argsValues := [][]Node{}

	for _, argNode := range node.Arguments {
		//argsIdentifiers = append(argsIdentifiers, funcDec.Arguments[i])
		argsValues = append(argsValues, []Node{argNode})
	}

	argsBody = append(argsBody, &VarDec{
		Identifier: argsIdentifiers,
		Value:      argsValues,
		Argument:   true,
		X:          node.X,
		Y:          node.Y,
	})

	addToScope := [][2]any{}

	body = slices.Concat(argsBody, body)
	if funcDec.Self != nil {
		addToScope = [][2]any{{selfKeyword, funcDec.Self}}
	}

	_, _, value := inter.CompeleteBody(body, true, false, addToScope...)

	return value
}

func (inter *Interpreter) CompeleteBody(body []Node, isFunc, isLoop bool, addToScope ...[2]any) (end, skip bool, value []any) {
	scope := NewScope(inter.CurrentScope)
	scope.IsFunc = isFunc
	scope.IsLoop = isLoop

	inter.Current(scope)
	defer inter.Current(scope.Parent)

	for _, addToScopeElem := range addToScope {
		scope.Add(addToScopeElem[0], addToScopeElem[1])
	}

	for _, node := range body {
		end, skip, valuesAny := inter.CompleteNode(node)
		if len(valuesAny) > 0 || end || skip {
			var values []any = nil

			if len(valuesAny) > 0 {
				values = []any{}

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

func (inter *Interpreter) SetTableElementValue(table *Map, keys []any, value any, index int) {
	if index >= len(keys) {
		return
	}

	key := keys[index]

	elem := table.GetElement(CL(key))
	switch elem := elem.Value.Get().(type) {
	case *Map:
		if index+1 >= len(keys) {
			table.Set(CL(key), CLPTR(value))
			break
		}
		inter.SetTableElementValue(elem, keys, value, index+1)
	default:
		table.Set(CL(key), CLPTR(value))
	}
}

func (inter *Interpreter) SetInstanceFieldValue(instance *StructObject, fields []string, value any, index int) {
	if index >= len(fields) {
		return
	}

	field := fields[index]

	fieldVal, _ := instance.Get(field)
	switch fieldVal := fieldVal.(type) {
	case *StructObject:
		if index >= len(fields) {
			instance.Set(field, value)
			break
		}

		inter.SetInstanceFieldValue(fieldVal, fields, value, index+1)
	default:
		instance.Set(field, value)
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
		throw("Attempt to index nothing", node.X, node.Y)
	}

	table := inter.GetNodeValue(tableNode)
	keys := []any{}
	for _, keyNode := range keyNodes {
		keys = append(keys, inter.GetNodeValue(keyNode))
	}

	value := inter.GetNodeValueS(node.Value, node.X, node.Y)

	switch table := table.(type) {
	case *Map:
		inter.SetTableElementValue(table, keys, value, 0)
	default:
		throw("Cannot index non-table value", node.X, node.Y)
	}
}

func (inter *Interpreter) SetFieldValue(node *SetFieldNode) {
	instanceNode, fieldNodes := inter.GetStructAndFieldNames(node.Field, []Node{})
	if instanceNode == nil {
		throw("Attempt to index nothing", node.X, node.Y)
	}

	instance := inter.GetNodeValue(instanceNode)
	fields := []string{}
	for _, fieldNode := range fieldNodes {
		fields = append(fields, fieldNode.(*IdentNode).Value)
	}

	value := inter.GetNodeValueS(node.Value, node.X, node.Y)

	switch instance := instance.(type) {
	case *StructObject:
		inter.SetInstanceFieldValue(instance, fields, value, 0)
	default:
		throw("Cannot assign field of non-instance value", node.X, node.Y)
	}
}

func (inter *Interpreter) DeclareStructure(structDecl *StructDeclNode) {
	identifier := structDecl.Identifier.Value
	fields := make([]*FieldDecl, len(structDecl.Fields))

	for i, fieldDeclNode := range structDecl.Fields {
		bits := int8(0)
		if fieldDeclNode.Bits != nil {
			bits = int8(fieldDeclNode.Bits.Value)
		}

		fields[i] = &FieldDecl{
			Identifier: fieldDeclNode.Identifier.Value,
			Method:     fieldDeclNode.Func != nil,
			Bits:       bits,
			Func:       fieldDeclNode.Func,
		}
	}

	if !inter.CurrentScope.Add(identifier, &Structure{
		Identifier: identifier,
		Fields:     fields,
	}) {
		throw("Attempt to declare the structure with the same name as the variable '%s'.", structDecl.X, structDecl.Y, identifier)
	}
}

func (inter *Interpreter) NewStructObject(structObjNode *StructNode) *StructObject {
	identifier := structObjNode.Identifier.Value

	originalStructure, ok := inter.CurrentScope.Get(identifier).(*Structure)
	if originalStructure == nil || !ok {
		throw("Attempt to make an instance of a nonexistent structure '%s'.", structObjNode.X, structObjNode.Y, identifier)
	}

	structObject := &StructObject{
		Identifier: identifier,
	}

	fields := []*Field{}

	for _, fieldDecl := range originalStructure.Fields {
		if fieldDecl.Func == nil {
			continue
		}
		fieldDecl.Func.Self = structObject

		cell := &Cell{}
		cell.Set(fieldDecl.Func)

		fields = append(fields, &Field{
			Identifier: fieldDecl.Identifier,
			Method:     fieldDecl.Method,
			Value:      cell,
		})
	}

	for i, fieldNode := range structObjNode.Fields {
		fieldName := fieldNode.Identifier.Value
		if !originalStructure.CheckField(fieldName) {
			throw("Attempt to assign a nonexistent field '%s' of structure '%s' while trying to make an instance.", structObjNode.X, structObjNode.Y, fieldName, identifier)
		}
		if originalStructure.IsAFunc(fieldName) {
			throw("Attempt to assign a value for a method '%s' of structure '%s'.", structObjNode.X, structObjNode.Y, fieldName, identifier)
		}

		v := inter.GetNodeValueS(fieldNode.Value, fieldNode.Identifier.X, fieldNode.Identifier.Y)

		cell := &Cell{}

		switch v := v.(type) {
		case []any:
			if len(v) > 1 {
				throw("Field cannot have more than one value.", fieldNode.Identifier.X, fieldNode.Identifier.Y)
			} else if len(v) == 0 {
				throw("Cannot assign a field cannot be an empty value.", fieldNode.Identifier.X, fieldNode.Identifier.Y)
			}

			cell.Set(v[0])
		default:
			cell.Set(v)
		}

		bits := originalStructure.Fields[i].Bits

		fields = append(fields, &Field{
			Identifier: fieldNode.Identifier.Value,
			LayoutType: bits,
			Value:      cell,
		})
	}

	structObject.Fields = fields
	structObject.ToMemoryLayout(structObject.Layout())

	return structObject
}

func (inter *Interpreter) CookValues(max_i int, values [][]Node, x, y int) []any {
	readyValues := []any{}

	for i := 0; i < max_i; i++ {
		if i >= len(values) {
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

	if len(readyValues) < max_i {
		readyValues = append(readyValues, make([]any, max_i-len(readyValues))...)
	}

	return readyValues
}

type ReturnNil struct{}

func (inter *Interpreter) CompleteNode(node Node) (end, skip bool, value []any) {
	scope := inter.CurrentScope
	switch node := node.(type) {
	case *FuncDec:
		if len(node.Identifier.Value) == 0 {
			throw("Name of the function cannot be empty.", node.X, node.Y)
		}
		if !inter.CurrentScope.Add(node.Identifier.Value, node) {
			throw("Attempt to redeclare a variable '%s'.", node.X, node.Y, node.Identifier.Value)
		}
	case *StructDeclNode:
		inter.DeclareStructure(node)
	case *VarDec:
		readyValues := inter.CookValues(len(node.Identifier), node.Value, node.X, node.Y)

		if len(readyValues) > len(node.Identifier) && !node.Argument {
			throw("Too many values for %d identifier(s).", node.X, node.Y, len(node.Identifier))
		} else if len(readyValues) > len(node.Identifier) && node.Argument {
			throw("Attempt to use multiple values as a single argument.", node.X, node.Y)
		}

		for i, ident := range node.Identifier {
			if !inter.CurrentScope.Add(ident.Value, readyValues[i]) {
				throw("Attempt to redeclare a variable '%s'.", node.X, node.Y, ident.Value)
			}
		}
	case *SetVar:
		readyValues := inter.CookValues(len(node.Value), node.Value, node.X, node.Y)

		if len(readyValues) > len(node.Var) {
			throw("Too many values in assignment", node.X, node.Y)
		} else if len(readyValues) < len(node.Var) {
			throw("Too few values in assignment", node.X, node.Y)
		}

		for i, ident := range node.Var {
			if !inter.CurrentScope.Set(ident.Value, readyValues[i], node.X, node.Y) {
				throw("Attempt to assign value to non-existing variable '%s'.", node.X, node.Y, node.Value)
			}
		}
	case *IndirAssignNode:
		valuePointerInterface := inter.GetNodeValue(node.Pointer)

		valuePointer, ok := valuePointerInterface.(uintptr)
		if !ok {
			throw("Attempt to do indirect assignment with invalid pointer.", node.X, node.Y)
		}

		pointerCell := scope.GetCellWithAddress(unsafe.Pointer(valuePointer))
		if pointerCell == nil {
			throw("Attempt to do indirect assignment of non-existing pointer.", node.X, node.Y)
		}

		valuePtr, ok := pointerCell.Get().(uintptr)
		if !ok {
			throw("Attempt to do indirect assignment with non-pointer value.", node.X, node.Y)
		}

		cellOfPtr := scope.GetCellWithAddress(unsafe.Pointer(valuePtr))
		if cellOfPtr == nil {
			throw("Attempt to do indirect assignment of non-existing value.", node.X, node.Y)
		}

		newValue := inter.GetNodeValueS(node.Value, node.X, node.Y)

		cellOfPtr.Set(newValue)
	case *FuncCall:
		inter.GetNodeValue(node)
	case *SetElem:
		inter.SetElementValue(node)
	case *IfStmt:
		result := inter.GetNodeValueS(node.Condition, node.X, node.Y)
		if result == true {
			return inter.CompeleteBody(node.Body, false, false)
		} else if result == false && node.Else != nil {
			return inter.CompleteNode(node.Else)
		}
	case *ElseStmt:
		if len(node.Condition) > 0 {
			result := inter.GetNodeValueS(node.Condition, node.X, node.Y)
			if result == true {
				return inter.CompeleteBody(node.Body, false, false)
			} else if result == false && node.Else != nil {
				return inter.CompleteNode(node.Else)
			}
		} else {
			return inter.CompeleteBody(node.Body, false, false)
		}
	case *ContinueNode:
		return false, true, nil
	case *BreakNode:
		return true, false, nil
	case *ReturnNode:
		var returnValue []any = nil
		if len(node.Value) > 0 {
			returnValue = []any{}
			for _, value := range node.Value {
				if len(value) > 0 {
					returnValue = append(returnValue, inter.GetNodeValueS(value, node.X, node.Y))
				} else {
					returnValue = append(returnValue, ReturnNil{})
				}
			}
		}

		return true, false, returnValue
	case *Import:
		{
			if inter.UnableToImport {
				throw("Import keyword must be at the beggining of the code.", node.Position(), node.Line())
			}
			if !inter.CurrentScope.MainScope {
				throw("Cannot use import keyword outside main scope.", node.Position(), node.Line())
			}

			if len(node.Path) > 0 && len(node.Path) < 2 {
				path, ok := node.Path[0].(*StrNode)
				if !ok {
					throw("Path for the import keyword cannot be a non-string value.", node.Position(), node.Line())
				}

				importModule(path.Value, inter.CurrentScope)
			} else if len(node.Path) > 1 {
				throw("Cannot import more than one file or module.", node.Position(), node.Line())
			} else {
				throw("Cannot import the file or the module without a path.", node.Position(), node.Line())
			}
			return false, false, nil
		}
	case *WhileNode:
		for inter.GetNodeValueS(node.Condition, node.X, node.Y) == true {
			end, skip, value := inter.CompeleteBody(node.Body, false, true)
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
				end, skip, returnValue := inter.CompeleteBody(node.Body, false, true, [2]any{keyIdent.Value, key}, [2]any{valueIdent.Value, value})

				if skip {
					continue
				} else if end || returnValue != nil {
					return end, skip, returnValue
				}
			}
		default:
			throw("Unable to iterate over a non-table value.", node.X, node.Y)
		}
	default:
		throw("Invalid node '%s'.", node.Position(), node.Line(), getInterfaceType(node))
	}
	inter.UnableToImport = true
	return false, false, nil
}

func (inter *Interpreter) Complete(logenv bool) map[any]*Cell {
	mainScope := NewScope(nil)
	mainScope.MainScope = true

	inter.CurrentScope = mainScope

	for ident, function := range builtinFuncs {
		mainScope.Add(ident, newFTemp(ident, function))
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
