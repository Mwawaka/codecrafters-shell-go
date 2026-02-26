package builtins

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

type History struct {
	commands []string
	mutex    sync.Mutex
}

func NewHistory() *History {
	return &History{
		commands: []string{},
	}
}

func (h *History) Add(cmd string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.commands = append(h.commands, cmd)
}

func (h *History) Display(args []string) (string, error) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if len(h.commands) == 0 {
		return "", nil

	}

	var builder strings.Builder
	commands := h.commands
	startIndex := 0

	if len(args) > 0 {
		if strings.HasPrefix(args[0], "-") {
			if len(args) < 2 {
				return "", fmt.Errorf("history: -r requires a filename")
			}

			switch args[0] {
			case "-r":
				if err := h.ReadHistoryFromFile(args[1]); err != nil {
					return "", err
				}

				return "", nil

			case "-w":
				if err := h.WriteHistoryToFile(args[1], false); err != nil {
					return "", err
				}

				return "", nil

			case "-a":
				if err := h.WriteHistoryToFile(args[1], true); err != nil {
					return "", err
				}

				return "", nil
			}

		}

		limit, err := strconv.Atoi(args[0])

		if err != nil {
			return "", err
		}

		if limit < len(h.commands) {
			startIndex = len(h.commands) - limit
			commands = commands[startIndex:]
		}
	}

	for i, cmd := range commands {
		builder.WriteString(fmt.Sprintf("%4d %s\n", startIndex+i+1, cmd))
	}

	return strings.TrimSuffix(builder.String(), "\n"), nil
}

func (h *History) ReadHistoryFromFile(filename string) error {
	file, err := os.Open(filename)

	if err != nil {
		return err
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		command := scanner.Text()

		// Skip empty lines
		if strings.TrimSpace(command) == "" {
			continue
		}

		h.commands = append(h.commands, command)
	}
	return scanner.Err()
}

func (h *History) WriteHistoryToFile(filename string, appendMode bool) error {
	var builder strings.Builder
	flags := os.O_WRONLY | os.O_CREATE

	if appendMode {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}

	file, err := os.OpenFile(filename, flags, 0644)

	if err != nil {
		return err
	}

	defer file.Close()

	for _, command := range h.commands {
		builder.WriteString(fmt.Sprintf("%s\n", command))
	}
	_, err = file.Write([]byte(builder.String()))
	return err
}
