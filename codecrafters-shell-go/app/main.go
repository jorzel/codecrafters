package main

import (
	"bufio"
	"fmt"
	"os"
	"slices"
	"strings"
)

// Ensures gofmt doesn't remove the "fmt" import in stage 1 (feel free to remove this!)
var _ = fmt.Fprint

const (
	EXIT = "exit"
	TYPE = "type"
	ECHO = "echo"
)

type handlerFunc = func([]string) string

var COMMANDS = []string{EXIT, TYPE, ECHO}

var HANDLERS = map[string]handlerFunc{
	EXIT: exitHandler,
	TYPE: typeHandler,
	ECHO: echoHandler,
}

func typeHandler(args []string) string {
	if len(args) == 0 {
		return fmt.Sprintf("type: missing argument")
	}
	cmd := args[0]
	if slices.Contains(COMMANDS, cmd) {
		return fmt.Sprintf("%s is a shell builtin", cmd)
	}
	return fmt.Sprintf("%s: not found\n", cmd)
}

func exitHandler(args []string) string {
	if len(args) == 0 || args[0] != "0" {
		return fmt.Sprintf("invalid arg for exit command, expected '0'")
	}
	os.Exit(0) // Exit the program with status code 0
	return ""
}

func echoHandler(args []string) string {
	return fmt.Sprintf("%s", strings.Join(args, " "))
}

func main() {
	// Uncomment this block to pass the first stage
	for {
		fmt.Fprint(os.Stdout, "$ ")

		// Wait for user input
		rawinput, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading input:", err)
			return
		}
		input := strings.Trim(rawinput, "\n\r ")
		cmd, args := eval(input)
		fn, ok := HANDLERS[cmd]
		if !ok {
			fmt.Printf("%s: command not found\n", cmd)
			continue
		}
		value := fn(args)
		fmt.Printf("%s\n", value)
	}
}

func eval(input string) (string, []string) {
	splited := strings.Split(input, " ")
	return splited[0], splited[1:]
}
