package main

import "testing"

func Test_tokenizer(t *testing.T) {
	val := tokenizer("echo")
	if val != "echo" {
		t.Error("incorrect result: expected echo, got ", val)
	}
}
