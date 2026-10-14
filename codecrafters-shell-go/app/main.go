package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
)

// Ensures gofmt doesn't remove the "fmt" import in stage 1 (feel free to remove this!)
var _ = fmt.Fprint

const (
	EXIT = "exit"
	TYPE = "type"
	ECHO = "echo"
)

type HandlerResult struct {
	text       string
	code       int
	shouldExit bool
}

type handlerFunc = func([]string) HandlerResult

var COMMANDS = []string{EXIT, TYPE, ECHO}

var HANDLERS = map[string]handlerFunc{
	EXIT: exitHandler,
	TYPE: typeHandler,
	ECHO: echoHandler,
}

func typeHandler(args []string) HandlerResult {
	if len(args) == 0 {
		return HandlerResult{
			text:       fmt.Sprintf("type: missing argument"),
			code:       1,
			shouldExit: false,
		}

	}
	cmd := args[0]
	if slices.Contains(COMMANDS, cmd) {
		return HandlerResult{
			text:       fmt.Sprintf("%s is a shell builtin", cmd),
			code:       0,
			shouldExit: false,
		}
	}
	return HandlerResult{
		text:       fmt.Sprintf("%s: not found", cmd),
		code:       1,
		shouldExit: false,
	}
}

func exitHandler(args []string) HandlerResult {
	code := 0
	if len(args) > 0 {
		var err error
		code, err = strconv.Atoi(args[0])
		if err != nil {
			return HandlerResult{
				text:       fmt.Sprintf("unexpected arg"),
				code:       1,
				shouldExit: true,
			}
		}
	}

	return HandlerResult{
		text:       "",
		code:       code,
		shouldExit: true,
	}
}

func echoHandler(args []string) HandlerResult {
	return HandlerResult{
		code:       0,
		text:       fmt.Sprintf("%s", strings.Join(args, " ")),
		shouldExit: false,
	}
}

func main() {
	os.Exit(run(os.Stdin, os.Stdout))
}

func run(input io.Reader, output io.Writer) int {
	reader := bufio.NewReader(input)
	for {
		fmt.Fprint(output, "$ ")

		// Wait for user input
		rawinput, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintln(output, "Error reading input:", err)
			return 1
		}
		input := strings.Trim(rawinput, "\n\r ")
		cmd, args := eval(input)
		fn, ok := HANDLERS[cmd]
		if !ok {
			fmt.Fprintf(output, "%s: command not found\n", cmd)
			continue
		}
		result := fn(args)
		if result.shouldExit {
			return result.code
		}
		fmt.Fprintf(output, "%s\n", result.text)
	}
}

func eval(input string) (string, []string) {
	splited := strings.Split(input, " ")
	return splited[0], splited[1:]
}
