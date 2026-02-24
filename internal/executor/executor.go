package executor

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/codecrafters-io/shell-starter-go/internal/builtins"
	"github.com/codecrafters-io/shell-starter-go/internal/redirect"
)

func Execute(pipeline [][]string, builtins map[string]builtins.CommandHandler) error {
	if len(pipeline) == 0 {
		return nil
	}

	var filename string
	lastCmd := pipeline[len(pipeline)-1]
	redirectIndex := -1
	appendMode := false
	fileDescriptor := redirect.FdStdout

	for i, token := range lastCmd {
		fd, isAppend, isRedirect := redirect.Redirect(token)

		if isRedirect {
			redirectIndex = i
			fileDescriptor = fd
			appendMode = isAppend
			break
		}
	}

	if redirectIndex != -1 && redirectIndex+1 < len(lastCmd) {
		filename = lastCmd[redirectIndex+1]
		pipeline[len(pipeline)-1] = append(lastCmd[:redirectIndex], lastCmd[redirectIndex+2:]...)
	}

	lastCmd = pipeline[len(pipeline)-1]
	command := lastCmd[0]
	args := lastCmd[1:]

	// Single command without pipeline
	if len(pipeline) == 1 {
		var buffer bytes.Buffer
		var stdout io.Writer = os.Stdout
		var stderr io.Writer = os.Stderr

		if handler, exists := builtins[command]; exists {
			out, err := handler(args)

			if err != nil {
				return err
			}

			if filename != "" {
				data := []byte(out + "\n")

				if fileDescriptor == redirect.FdStderr {
					data = []byte{}
					os.Stdout.WriteString(out + "\n")
				}
				return writeToFile(filename, data, appendMode)
			}

			fmt.Fprintln(os.Stdout, out)

			return nil
		}

		if filename != "" {
			switch fileDescriptor {
			case redirect.FdStdout:
				stdout = &buffer
			case redirect.FdStderr:
				stderr = &buffer
			}
		}

		err := runExternal(command, args, stdout, stderr)

		if filename != "" {
			return writeToFile(filename, buffer.Bytes(), appendMode)
		}

		return err
	}

	// Multiple command handling
	var cmds []*exec.Cmd
	var previousRead *os.File

	// Start all commands except from the last
	for i := 0; i < len(pipeline)-1; i++ {
		subCmd := pipeline[i]
		command := subCmd[0]
		args := subCmd[1:]
		cmd := exec.Command(command, args...)

		if previousRead != nil {
			cmd.Stdin = previousRead
		}

		r, w, err := os.Pipe()

		if err != nil {
			return err
		}

		cmd.Stdout = w
		cmd.Stderr = os.Stderr

		if err := cmd.Start(); err != nil {
			return err
		}

		cmds = append(cmds, cmd)
		w.Close()
		previousRead = r
	}

	// Handling last command
	cmd := exec.Command(command, args...)

	if previousRead != nil {
		cmd.Stdin = previousRead
	}

	cmd.Stderr = os.Stderr
	var buffer bytes.Buffer

	if filename != "" {
		switch fileDescriptor {
		case redirect.FdStdout:
			cmd.Stdout = &buffer
		case redirect.FdStderr:
			cmd.Stderr = &buffer
			cmd.Stdout = os.Stdout
		}
	} else {
		cmd.Stdout = os.Stdout
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	cmds = append(cmds, cmd)

	if previousRead != nil {
		previousRead.Close()
	}

	var lastError error

	for _, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			lastError = err
		}
	}

	if filename != "" {
		if err := writeToFile(filename, buffer.Bytes(), appendMode); err != nil {
			return err
		}
	}

	return lastError
}

func runExternal(cmdName string, args []string, stdout, stderr io.Writer) error {
	cmd := exec.Command(cmdName, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
