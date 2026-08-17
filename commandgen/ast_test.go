package commandgen

import (
	"strings"
	"testing"
)

const fixtureGombitMain = `package main

import (
	"context"
	"os"

	"github.com/LAA-Software-Engineering/gombit/cli"

	"github.com/example/demo/internal/product"
)

func main() {
	root := cli.NewRoot(os.Stdout, os.Stderr)
	product.RegisterCommands(root)
	if err := cli.ExecuteRoot(context.Background(), root, os.Args[1:]); err != nil {
		os.Exit(1)
	}
}
`

const fixtureRegister = `package commands

import "github.com/LAA-Software-Engineering/gombit/cli"

func Register(root *cli.Command) {
}
`

func TestAddImportAndRegisterIdempotent(t *testing.T) {
	first, err := AddImportAndRegister([]byte(fixtureGombitMain), "github.com/example/demo/internal/commands", "commands")
	if err != nil {
		t.Fatalf("AddImportAndRegister: %v", err)
	}
	src := string(first)
	if !strings.Contains(src, "commands.Register(root)") {
		t.Fatalf("missing commands.Register:\n%s", src)
	}
	if !strings.Contains(src, "product.RegisterCommands(root)") {
		t.Fatalf("lost product.RegisterCommands:\n%s", src)
	}

	second, err := AddImportAndRegister(first, "github.com/example/demo/internal/commands", "commands")
	if err != nil {
		t.Fatalf("second AddImportAndRegister: %v", err)
	}
	count, err := CountRegisterCalls(second, "commands")
	if err != nil {
		t.Fatalf("CountRegisterCalls: %v", err)
	}
	if count != 1 {
		t.Fatalf("duplicated Register: %d\n%s", count, second)
	}
}

func TestAddCommandCallIdempotent(t *testing.T) {
	first, err := AddCommandCall([]byte(fixtureRegister), "NewGreetCommand")
	if err != nil {
		t.Fatalf("AddCommandCall: %v", err)
	}
	if !strings.Contains(string(first), "cli.AddCommand(root, NewGreetCommand())") {
		t.Fatalf("missing AddCommand:\n%s", first)
	}

	second, err := AddCommandCall(first, "NewGreetCommand")
	if err != nil {
		t.Fatalf("second AddCommandCall: %v", err)
	}
	count, err := CountConstructorCalls(second, "NewGreetCommand")
	if err != nil {
		t.Fatalf("CountConstructorCalls: %v", err)
	}
	if count != 1 {
		t.Fatalf("duplicated constructor: %d\n%s", count, second)
	}

	third, err := AddCommandCall(second, "NewHelloCommand")
	if err != nil {
		t.Fatalf("AddCommandCall hello: %v", err)
	}
	greet, err := CountConstructorCalls(third, "NewGreetCommand")
	if err != nil {
		t.Fatalf("CountConstructorCalls greet: %v", err)
	}
	hello, err := CountConstructorCalls(third, "NewHelloCommand")
	if err != nil {
		t.Fatalf("CountConstructorCalls hello: %v", err)
	}
	if greet != 1 || hello != 1 {
		t.Fatalf("greet=%d hello=%d\n%s", greet, hello, third)
	}
}

func TestAddImportAndRegisterRequiresRegistrationPoint(t *testing.T) {
	src := []byte("package main\n\nfunc main() {}\n")
	_, err := AddImportAndRegister(src, "github.com/example/demo/internal/commands", "commands")
	if err == nil || !strings.Contains(err.Error(), "registration point") {
		t.Fatalf("error = %v, want registration point", err)
	}
}
