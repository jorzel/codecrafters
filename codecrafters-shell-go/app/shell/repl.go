package shell

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const exitFailure = 1

type handlerResult struct {
	code       int
	shouldExit bool
}

type handlerFunc = func([]string, io.Writer, io.Writer) handlerResult

type ExecutableRunner interface {
	Run(path, name string, args []string, stdout, stderr io.Writer) error
}

type ExecutableFinder interface {
	Find(name string) (string, bool)
}

type Shell struct {
	executableFinder ExecutableFinder
	executableRunner ExecutableRunner
	handlers         map[string]handlerFunc
}

func New(executableFinder ExecutableFinder, executableRunner ExecutableRunner) *Shell {
	sh := &Shell{
		executableFinder: executableFinder,
		executableRunner: executableRunner,
	}

	sh.handlers = map[string]handlerFunc{
		"exit": exitHandler,
		"type": sh.typeHandler,
		"echo": echoHandler,
	}
	return sh
}

func (sh *Shell) Run(stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	reader := bufio.NewReader(stdin)
	for {
		fmt.Fprint(stdout, "$ ")

		// Wait for user input
		rawinput, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(stdout, "Error reading input: %s\n", err)
			return exitFailure
		}
		input := strings.Trim(rawinput, "\n\r ")
		cmd, args := eval(input)

		var result handlerResult
		fn, ok := sh.handlers[cmd]
		if !ok {
			result = sh.fallbackHandler(cmd, args, stdout, stderr)
		} else {
			result = fn(args, stdout, stderr)
		}

		if result.shouldExit {
			return result.code
		}
	}
}

func (sh *Shell) fallbackHandler(cmd string, args []string, stdout io.Writer, stderr io.Writer) handlerResult {
	fullPath, ok := sh.executableFinder.Find(cmd)
	if !ok {
		fmt.Fprintf(stderr, "%s: command not found\n", cmd)
		return handlerResult{}
	}
	err := sh.executableRunner.Run(fullPath, cmd, args, stdout, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "%s: %s\n", cmd, err)
	}
	return handlerResult{}
}

func (sh *Shell) typeHandler(args []string, stdout io.Writer, stderr io.Writer) handlerResult {
	if len(args) == 0 {
		fmt.Fprintf(stderr, "type: missing argument\n")
		return handlerResult{
			code: exitFailure,
		}

	}
	cmd := args[0]
	_, ok := sh.handlers[cmd]
	if ok {
		fmt.Fprintf(stdout, "%s is a shell builtin\n", cmd)
		return handlerResult{}
	}
	fullPath, ok := sh.executableFinder.Find(cmd)
	if ok {
		fmt.Fprintf(stdout, "%s is %s\n", cmd, fullPath)
		return handlerResult{}
	}
	fmt.Fprintf(stderr, "%s: not found\n", cmd)
	return handlerResult{
		code: exitFailure,
	}

}

func eval(input string) (string, []string) {
	splited := strings.Split(input, " ")
	return splited[0], splited[1:]
}

func exitHandler(args []string, stdout io.Writer, stderr io.Writer) handlerResult {
	code := 0
	if len(args) > 0 {
		var err error
		code, err = strconv.Atoi(args[0])
		if err != nil {
			fmt.Fprintf(stdout, "unexpected arg\n")
			return handlerResult{
				code:       exitFailure,
				shouldExit: true,
			}
		}
	}

	return handlerResult{
		code:       code,
		shouldExit: true,
	}
}

func echoHandler(args []string, stdout io.Writer, stderr io.Writer) handlerResult {
	fmt.Fprintf(stdout, "%s\n", strings.Join(args, " "))
	return handlerResult{}
}
