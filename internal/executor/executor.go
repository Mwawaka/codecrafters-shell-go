package executor

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/codecrafters-io/shell-starter-go/internal/builtins"
	"github.com/codecrafters-io/shell-starter-go/internal/redirect"
)

type CommandRunner struct {
	Name     string
	Args     []string
	Stdin    io.Reader
	Stdout   io.Writer
	Stderr   io.Writer
	Builtins map[string]builtins.CommandHandler
}

func (cr *CommandRunner) run() error {
	if handler, exists := cr.Builtins[cr.Name]; exists {
		output, err := handler(cr.Args)

		if err != nil {
			return err
		}

		_, err = cr.Stdout.Write([]byte(output + "\n"))
		return err
	}

	cmd := exec.Command(cr.Name, cr.Args...)
	cmd.Stdin = cr.Stdin
	cmd.Stdout = cr.Stdout
	cmd.Stderr = cr.Stderr
	return cmd.Run()
}

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
		var stdin io.Reader = os.Stdin
		var stdout io.Writer = os.Stdout
		var stderr io.Writer = os.Stderr

		runner := &CommandRunner{
			Name:     command,
			Args:     args,
			Stdin:    stdin,
			Stdout:   stdout,
			Stderr:   stderr,
			Builtins: builtins,
		}

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

		err := runner.run()

		if filename != "" {
			return writeToFile(filename, buffer.Bytes(), appendMode)
		}

		return err
	}

	// Multiple command handling
	var pipes []*os.File
	var wg sync.WaitGroup
	var errors []error
	var errorsMutex sync.Mutex

	addError := func(err error) {
		errorsMutex.Lock()
		errors = append(errors, err)
		errorsMutex.Unlock()
	}

	numCommands := len(pipeline)
	pipeReaders := make([]*os.File, numCommands-1)
	pipeWriters := make([]*os.File, numCommands-1)

	for i := 0; i < numCommands-1; i++ {
		r, w, err := os.Pipe()

		if err != nil {
			return err
		}

		pipeReaders[i] = r
		pipeWriters[i] = w
		pipes = append(pipes, r, w)
	}

	defer func() {
		for _, p := range pipes {
			p.Close()
		}
	}()

	for i := 0; i < numCommands; i++ {
		var stdin io.Reader = os.Stdin
		var stdout io.Writer = os.Stdout
		var buffer *bytes.Buffer
		var pipeWriter *os.File

		subCmd := pipeline[i]
		command := subCmd[0]
		args := subCmd[1:]

		if i > 0 {
			stdin = pipeReaders[i-1]
		}

		if i < numCommands-1 {
			stdout = pipeWriters[i]
			pipeWriter = pipeWriters[i]
		} else {
			if filename != "" {
				buffer = &bytes.Buffer{}
				switch fileDescriptor {
				case redirect.FdStdout:
					stdout = buffer
				case redirect.FdStderr:
					stdout = os.Stdout
				}
			}
		}

		wg.Add(1)

		go func(name string, args []string, stdin io.Reader, stdout io.Writer, buf *bytes.Buffer, isLast bool, writer *os.File) {
			defer wg.Done()

			runner := &CommandRunner{
				Name:     name,
				Args:     args,
				Stdin:    stdin,
				Stdout:   stdout,
				Stderr:   os.Stderr,
				Builtins: builtins,
			}

			if err := runner.run(); err != nil {
				addError(err)
			}

			// Close pipe writer immediately after command finishes
			if writer != nil {
				writer.Close()
			}

			// Last command with redirection, write to file
			if isLast && buf != nil && filename != "" {
				if err := writeToFile(filename, buf.Bytes(), appendMode); err != nil {
					addError(err)
				}
			}
		}(command, args, stdin, stdout, buffer, i == numCommands-1)
	}

	wg.Wait()

	if len(errors) > 0 {
		return errors[0]
	}

	return nil
}
