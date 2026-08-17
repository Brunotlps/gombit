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

const fixtureCommands = `package commands

import "github.com/LAA-Software-Engineering/gombit/cli"

func RegisterCommands(root *cli.Command) {
}
`

const fixtureFrameworkMain = `package main

import (
	"context"
	"io"

	"github.com/LAA-Software-Engineering/gombit/cli"
)

func run(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) error {
	return cli.Execute(ctx, args, stdout, stderr)
}
`

func TestAddImportAndRegisterIdempotent(t *testing.T) {
	first, err := AddImportAndRegister([]byte(fixtureGombitMain), "github.com/example/demo/internal/commands", "commands")
	if err != nil {
		t.Fatalf("AddImportAndRegister: %v", err)
	}
	src := string(first)
	if !strings.Contains(src, "commands.RegisterCommands(root)") {
		t.Fatalf("missing commands.RegisterCommands:\n%s", src)
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
		t.Fatalf("duplicated RegisterCommands: %d\n%s", count, second)
	}
}

func TestAddCommandCallIdempotent(t *testing.T) {
	first, err := AddCommandCall([]byte(fixtureCommands), "NewGreetCommand")
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

func TestAddImportAndRegisterDoesNotUseExecute(t *testing.T) {
	_, err := AddImportAndRegister([]byte(fixtureFrameworkMain), "github.com/example/demo/internal/commands", "commands")
	if err == nil || !strings.Contains(err.Error(), "registration point") {
		t.Fatalf("error = %v, want registration point (cli.Execute is not an anchor)", err)
	}
}

func TestAddImportAndRegisterAfterNewRoot(t *testing.T) {
	src := []byte(`package main

import (
	"os"

	"github.com/LAA-Software-Engineering/gombit/cli"
)

func main() {
	root := cli.NewRoot(os.Stdout, os.Stderr)
	_ = root
}
`)
	got, err := AddImportAndRegister(src, "github.com/example/demo/internal/book", "book")
	if err != nil {
		t.Fatalf("AddImportAndRegister: %v", err)
	}
	if !strings.Contains(string(got), "book.RegisterCommands(root)") {
		t.Fatalf("missing book.RegisterCommands after NewRoot:\n%s", got)
	}
}

func TestAddCommandCallRequiresRegisterCommands(t *testing.T) {
	src := []byte("package book\n\nfunc Register(app int) {}\n")
	_, err := AddCommandCall(src, "NewGreetCommand")
	if err == nil || !strings.Contains(err.Error(), "RegisterCommands") {
		t.Fatalf("error = %v, want RegisterCommands", err)
	}
}

func TestFormatGoFailsOnInvalidSource(t *testing.T) {
	_, err := formatGo("package commands\nfunc {")
	if err == nil {
		t.Fatal("expected format error")
	}
	if !strings.Contains(err.Error(), "format source") {
		t.Fatalf("error = %v, want format source", err)
	}
}
