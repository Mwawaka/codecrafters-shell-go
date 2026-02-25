package builtins

import (
	"fmt"
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

	commands := h.commands
	if len(h.commands) == 0 {
		return "", nil

	}

	var builder strings.Builder

	if len(args) > 0 {
		limit, err := strconv.Atoi(args[0])
		if err != nil {
			return "", err
		}
		commands = commands[limit:]
	}

	for i, cmd := range commands {
		builder.WriteString(fmt.Sprintf("%4d %s\n", i+1, cmd))
	}

	return strings.TrimSuffix(builder.String(), "\n"), nil
}
