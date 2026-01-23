package main

import (
	"os"
	"testing"
)

func TestMain_Help(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"faro", "--help"}
	main()
}
