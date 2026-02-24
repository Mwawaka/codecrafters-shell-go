package parser

import (
	"fmt"
	"strings"
	"unicode"
)

func Parse(command string) ([]string, error) {
	runes := []rune(command)
	var builder strings.Builder
	tokens := []string{}
	inSingleQuote := false
	inDoubleQuote := false
	inBackSlash := false

	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if inBackSlash {
			if inDoubleQuote && !isEscapableInDoubleQuote(r) {
				builder.WriteRune('\\')
			}

			builder.WriteRune(r)
			inBackSlash = false
			continue
		}

		if r == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
			continue
		}

		if r == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
			continue
		}

		if r == '\\' && !inSingleQuote {
			inBackSlash = true
			continue
		}

		if r == '>' && !inSingleQuote && !inDoubleQuote {
			nextRune, hasNext := peekNext(runes, i)
			currentToken := builder.String()

			if len(currentToken) == 1 && currentToken[0] >= '0' && currentToken[0] <= '9' {
				builder.Reset()

				if hasNext && nextRune == '>' {
					tokens = append(tokens, currentToken+">>")
					i++
				} else {
					tokens = append(tokens, currentToken+">")
				}
				continue
			}

			if hasNext && nextRune == '>' {
				tokens = flush(&builder, tokens)
				tokens = append(tokens, ">>")
				i++
				continue
			}

			tokens = flush(&builder, tokens)
			tokens = append(tokens, ">")
			builder.Reset()
			continue
		}

		if unicode.IsSpace(r) && !inSingleQuote && !inDoubleQuote {
			tokens = flush(&builder, tokens)
			continue
		}

		builder.WriteRune(r)
	}

	if inSingleQuote || inDoubleQuote {
		return nil, fmt.Errorf("unclosed quote")
	}

	tokens = flush(&builder, tokens)
	// fmt.Println("Tokens: ", tokens)
	return tokens, nil
}

func peekNext(runes []rune, i int) (rune, bool) {
	nextIndx := i + 1
	if nextIndx < len(runes) {
		return runes[nextIndx], true
	}
	return 0, false
}

func flush(b *strings.Builder, tokens []string) []string {
	if b.Len() > 0 {
		tokens = append(tokens, b.String())
		b.Reset()
	}

	return tokens
}

func isEscapableInDoubleQuote(r rune) bool {
	return r == '"' || r == '\\' || r == '$' || r == '`' || r == '\n'
}
