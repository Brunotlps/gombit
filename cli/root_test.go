package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestAddCommandAppearsInHelpAndRuns(t *testing.T) {
	attach := func(stdout, stderr *bytes.Buffer) *Command {
		root := NewRoot(stdout, stderr)
		AddCommand(root, &Command{
			Use:   "greet",
			Short: "test greet",
			RunE: func(cmd *Command, args []string) error {
				_, err := cmd.OutOrStdout().Write([]byte("greet: ok\n"))
				return err
			},
		})
		return root
	}

	helpOut := new(bytes.Buffer)
	if err := ExecuteRoot(context.Background(), attach(helpOut, new(bytes.Buffer)), []string{"--help"}); err != nil {
		t.Fatalf("help: %v", err)
	}
	if !strings.Contains(helpOut.String(), "greet") {
		t.Fatalf("help missing greet:\n%s", helpOut.String())
	}

	runOut := new(bytes.Buffer)
	if err := ExecuteRoot(context.Background(), attach(runOut, new(bytes.Buffer)), []string{"greet"}); err != nil {
		t.Fatalf("greet: %v", err)
	}
	if !strings.Contains(runOut.String(), "greet: ok") {
		t.Fatalf("greet output = %q", runOut.String())
	}
}

func TestAddCommandNilRootPanics(t *testing.T) {
	defer func() {
		got := recover()
		if got == nil {
			t.Fatal("expected panic")
		}
		msg, ok := got.(string)
		if !ok || !strings.Contains(msg, "nil root") {
			t.Fatalf("panic = %#v, want nil root", got)
		}
	}()
	AddCommand(nil, &Command{Use: "x"})
}
