package autocompleter

import (
	"fmt"
	"os"
	"runtime"
	"slices"
	"strings"
)

type TabCompleter struct {
	tabCount      int
	lastInput     string
	cachedMatches []string
	builtinNames  []string
}

func (t *TabCompleter) Do(line []rune, pos int) (newLine [][]rune, length int) {
	var isCommandPosition bool
	var lastWord string

	currentInput := string(line[:pos])

	if currentInput != t.lastInput {
		t.tabCount = 0
		t.lastInput = currentInput
		t.cachedMatches = nil
	}

	words := strings.Fields(currentInput)

	if len(words) == 0 {
		isCommandPosition = true
		lastWord = ""
	} else if len(words) == 1 && !strings.HasSuffix(currentInput, " ") {
		isCommandPosition = true
		lastWord = words[0]
	} else {
		isCommandPosition = false

		if strings.HasSuffix(currentInput, " ") {
			lastWord = ""
		} else {
			lastWord = words[len(words)-1]
		}
	}

	t.tabCount++

	if t.tabCount == 1 {
		var matches []string

		if isCommandPosition {
			builtinMatches := t.listCommands(lastWord)
			executableMatches := listExecutables(lastWord)
			matches = append(builtinMatches, executableMatches...)
		} else {
			matches = listFiles(lastWord)
		}

		t.cachedMatches = matches

		if len(matches) == 1 {
			completion := matches[0]

			if !strings.HasSuffix(completion, "/") {
				completion += " "
			}

			return [][]rune{[]rune(completion[len(lastWord):])}, len(lastWord)
		} else if len(matches) > 1 {
			completion := lcp(matches)
			return [][]rune{[]rune(completion[len(lastWord):])}, len(lastWord)
		} else {
			os.Stdout.Write([]byte("\x07"))
			os.Stdout.Sync()
			return [][]rune{}, len(lastWord)
		}
	}

	if t.tabCount == 2 {
		if len(t.cachedMatches) > 0 {
			slices.Sort(t.cachedMatches)
			fmt.Fprintf(os.Stdout, "\n%s\n", strings.Join(t.cachedMatches, "  "))
		}
		t.tabCount = 0
		t.cachedMatches = nil
		return [][]rune{[]rune("")}, len(lastWord)
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

	// sort.Strings(strs)
	slices.Sort(strs) // more ergonomic compared to sort package

	firstString := []rune(strs[0])
	lastString := []rune(strs[len(strs)-1])

	i := 0

	for i < len(firstString) && i < len(lastString) && firstString[i] == lastString[i] {
		i++
	}

	return string(firstString[:i])
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

			// check executable (Unix only - Windows all files are "executable" )
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

func listFiles(prefix string) []string {
	var matches []string

	// Read current directory
	entries, err := os.ReadDir(".")

	if err != nil {
		return matches
	}

	for _, entry := range entries {
		name := entry.Name()

		//Skip hidden files
		if strings.HasPrefix(name, ".") {
			continue
		}

		// Check if matches prefix
		if strings.HasPrefix(name, prefix) {
			// Add a trailing / for directories
			if entry.IsDir() {
				name += "/"
			}

			matches = append(matches, name)
		}
	}

	return matches
}

func beep(matches []string) {
	if len(matches) == 0 {
		os.Stdout.Write([]byte("\x07"))
		os.Stdout.Sync()
	}
}
