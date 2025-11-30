package main

import (
	"os"
	"path/filepath"
)

const (
	plName           = "Yarik#"
	shortennedPLName = "yks"

	fileType            = ".yks"
	major, minor, patch = 0, 7, 4
	stage               = "beta"
)

var (
	args     = os.Args[1:]
	commands = make(map[string]func(args []string))
	libs     = filepath.Join(getParentPath(getParentPath(getSelfPath())), "src")
)
