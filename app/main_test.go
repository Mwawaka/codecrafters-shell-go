package main

import (
	"testing"

	"github.com/codecrafters-io/shell-starter-go/internal/parser"
)

func Test_tokenizer(t *testing.T) {
	val, _ := parser.Parse("echo")
	if len(val) == 0 {
		t.Error("incorrect result: expected echo, got ", val)
	}
}