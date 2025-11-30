package main

func floatIsInt(f float64) bool
func argsCheck(v []any, min, max int, expectedDataTypes ...string)
func numtostr(v any) string
func numberToFloat64(n any) (float64, bool)
func numberToInt(n any) (int64, bool)
func mustNTOF64(n any) float64
func getValueType(v any) string
func checkDataType(expected string, v any) bool
func getSelfPath() string
func help([]string)
func handle(err error)
func getFileString(path string) (string, error)
func throw(errForm string, x, y int, v ...any)
func throwNoPos(errForm string, v ...any)
func getParentPath(path string) string
func getAbsPath(relPath string) string
func run(fileAbs, fileRel string, info bool) map[any]*Cell
