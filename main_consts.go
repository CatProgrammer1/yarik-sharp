package main

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	plName           = "Yarik#"
	shortennedPLName = "yks"

	fileType            = ".yks"
	major, minor, patch = 0, 1, 0
	stage               = "beta"
)

var (//go run yks run test.yks
	args     = os.Args[1:]
	commands = make(map[string]func(args []string))
	libs     = filepath.Join(getParentPath(getParentPath(getSelfPath())), "src")

	println = fmt.Println
)
