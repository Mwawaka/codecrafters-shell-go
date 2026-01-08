package autocompleter

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
)

type TabCompleter struct {
	tabCount      int
	lastInput     string
	cachedMatches []string
	builtinNames  []string
}

func (t *TabCompleter) Do(line []rune, pos int) (newLine [][]rune, length int) {
	currentInput := string(line[:pos])

	// Reset count if input changed
	if currentInput != t.lastInput {
		t.tabCount = 0
		t.lastInput = currentInput
		t.cachedMatches = nil
	}

	t.tabCount++

	if t.tabCount == 1 {
		// Check builtins first
		builtinMatches := t.listCommands(currentInput)
		executableMatches := listExecutables(currentInput)
		t.cachedMatches = executableMatches

		if len(builtinMatches) == 1 {
			// Autocomplete single builtin
			completion := builtinMatches[0] + " "
			return [][]rune{[]rune(completion[len(currentInput):])}, len(currentInput)
		}

		if len(executableMatches) == 1 {
			// Autocomplete single builtin
			completion := executableMatches[0] + " "
			return [][]rune{[]rune(completion[len(currentInput):])}, len(currentInput)
		} else if len(executableMatches) > 1 {
			completion := lcp(executableMatches)
			return [][]rune{[]rune(completion[len(currentInput):])}, len(currentInput)
		} else {
			// No single builtin match: ring bell
			os.Stdout.Write([]byte("\x07"))
			os.Stdout.Sync()
			return [][]rune{}, len(currentInput)
		}
	}

	if t.tabCount == 2 {
		// Show executable matches

		if len(t.cachedMatches) > 0 {
			sort.Strings(t.cachedMatches)
			fmt.Fprintf(os.Stdout, "\n%s\n", strings.Join(t.cachedMatches, "  "))
		}

		t.tabCount = 0
		t.cachedMatches = nil
		return [][]rune{[]rune("")}, len(currentInput)
	}

	return nil, len(currentInput)
}

func NewTabCompleter(builtinNames []string) *TabCompleter {
	return &TabCompleter{
		builtinNames: builtinNames,
	}
}

func lcp(strs []string) string {

	if len(strs) == 0 {
		return ""
	}

	if len(strs) == 1 {
		return strs[0]
	}

	sort.Strings(strs)

	firstString := strs[0]
	lastString := strs[len(strs)-1]

	i := 0

	for i < len(firstString) && i < len(lastString) && firstString[i] == lastString[i] {
		i++
	}

	return firstString[:i]
}

func (t *TabCompleter) listCommands(prefix string) []string {
	var matches []string

	for _, cmd := range t.builtinNames {
		if strings.HasPrefix(cmd, prefix) {
			matches = append(matches, cmd)
		}
	}

	beep(matches)
	return matches
}

func listExecutables(prefix string) []string {
	var matches []string
	seen := make(map[string]bool)
	pathEnv := os.Getenv("PATH")

	if pathEnv == "" {
		return nil
	}

	separator := ":" //Unix based systems

	if runtime.GOOS == "windows" {
		separator = ";"
	}

	dirs := strings.Split(pathEnv, separator)

	for _, dir := range dirs {
		if dir == "" {
			continue
		}

		entries, err := os.ReadDir(dir)

		if err != nil {
			continue // skips bad dirs
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue // skips subdirs
			}

			name := entry.Name()

			// no auto completion for hidden files by default
			if strings.HasPrefix(name, ".") {
				continue
			}

			// check executable (Unix only - Windows alll files are "executable" )
			if runtime.GOOS != "windows" {
				info, err := entry.Info()

				if err != nil {
					continue
				}

				// check if file is executable
				if info.Mode()&0111 == 0 {
					continue
				}
			}

			if strings.HasPrefix(name, prefix) && !seen[name] {
				matches = append(matches, name)
				seen[name] = true
			}
		}
	}

	beep(matches)
	return matches
}

func beep(matches []string) {
	if len(matches) == 0 {
		os.Stdout.Write([]byte("\x07"))
		os.Stdout.Sync()
	}
}
