package shell

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const exitFailure = 1

type ExecutableFinder interface {
	Find(name string) (string, bool)
}

type Shell struct {
	executableFinder ExecutableFinder
	handlers         map[string]handlerFunc
}

func New(executableFinder ExecutableFinder) *Shell {
	sh := &Shell{
		executableFinder: executableFinder,
	}

	sh.handlers = map[string]handlerFunc{
		"exit": exitHandler,
		"type": sh.typeHandler,
		"echo": echoHandler,
	}
	return sh
}

func (sh *Shell) Run(input io.Reader, output io.Writer) int {
	reader := bufio.NewReader(input)
	for {
		fmt.Fprint(output, "$ ")

		// Wait for user input
		rawinput, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintln(output, "Error reading input:", err)
			return exitFailure
		}
		input := strings.Trim(rawinput, "\n\r ")
		cmd, args := eval(input)
		fn, ok := sh.handlers[cmd]
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

func (sh *Shell) typeHandler(args []string) handlerResult {
	if len(args) == 0 {
		return handlerResult{
			text: fmt.Sprintf("type: missing argument"),
			code: exitFailure,
		}

	}
	cmd := args[0]
	_, ok := sh.handlers[cmd]
	if ok {
		return handlerResult{
			text: fmt.Sprintf("%s is a shell builtin", cmd),
		}
	}
	fullPath, ok := sh.executableFinder.Find(cmd)
	if ok {
		return handlerResult{
			text: fmt.Sprintf("%s is %s", cmd, fullPath),
		}
	}
	return handlerResult{
		text: fmt.Sprintf("%s: not found", cmd),
		code: exitFailure,
	}

}

func eval(input string) (string, []string) {
	splited := strings.Split(input, " ")
	return splited[0], splited[1:]
}

type handlerResult struct {
	text       string
	code       int
	shouldExit bool
}

type handlerFunc = func([]string) handlerResult

func exitHandler(args []string) handlerResult {
	code := 0
	if len(args) > 0 {
		var err error
		code, err = strconv.Atoi(args[0])
		if err != nil {
			return handlerResult{
				text:       fmt.Sprintf("unexpected arg"),
				code:       exitFailure,
				shouldExit: true,
			}
		}
	}

	return handlerResult{
		text:       "",
		code:       code,
		shouldExit: true,
	}
}

func echoHandler(args []string) handlerResult {
	return handlerResult{
		text: fmt.Sprintf("%s", strings.Join(args, " ")),
	}
}
