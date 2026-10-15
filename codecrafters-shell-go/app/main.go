package main

import (
	"os"

	"github.com/codecrafters-io/shell-starter-go/app/pathlookup"
	"github.com/codecrafters-io/shell-starter-go/app/process"
	"github.com/codecrafters-io/shell-starter-go/app/shell"
)

func main() {
	sh := shell.New(pathlookup.New(os.Getenv("PATH")), process.New())
	os.Exit(sh.Run(os.Stdin, os.Stdout, os.Stderr))
}
