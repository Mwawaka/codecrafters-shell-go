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

func Execute(command, filename string, args []string, builtins map[string]builtins.CommandHandler, fileDescriptor int, appendMode bool) error {
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

func runExternal(cmdName string, args []string, stdout, stderr io.Writer) error {
	cmd := exec.Command(cmdName, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
