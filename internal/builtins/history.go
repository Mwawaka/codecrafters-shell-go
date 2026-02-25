package builtins

import (
	"fmt"
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

	for i, cmd := range h.commands {
		builder.WriteString(fmt.Sprintf("%4d %s", i+1, cmd))
	}

	return strings.TrimSuffix(builder.String(), "\n"), nil
}
