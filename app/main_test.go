package main

import "testing"

func Test_tokenizer(t *testing.T) {
	val := tokenizer("echo")
	if len(val) == 0 {
		t.Error("incorrect result: expected echo, got ", val)
	}
}
