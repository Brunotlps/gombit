package commandgen

import (
	"fmt"
	"go/format"
	"strings"
)

type fileSpec struct {
	relPath string
	content []byte
	owned   bool
}

func renderCommandFile(name CommandName) ([]byte, error) {
	var b strings.Builder
	b.WriteString("// " + GeneratedBanner + "\n")
	b.WriteString("package " + name.Package + "\n\n")
	b.WriteString("import (\n")
	b.WriteString("\t\"fmt\"\n\n")
	b.WriteString("\t\"github.com/gombit-dev/gombit/cli\"\n")
	b.WriteString(")\n\n")
	b.WriteString("// " + name.Constructor + " returns the " + name.Use + " management command.\n")
	b.WriteString("// Attach it with cli.AddCommand from RegisterCommands (D13 / ADR-014).\n")
	b.WriteString("func " + name.Constructor + "() *cli.Command {\n")
	b.WriteString("\treturn &cli.Command{\n")
	b.WriteString("\t\tUse:   \"" + name.Use + "\",\n")
	b.WriteString("\t\tShort: \"Run the " + name.Use + " management command\",\n")
	b.WriteString("\t\tRunE: func(cmd *cli.Command, args []string) error {\n")
	b.WriteString("\t\t\t_, err := fmt.Fprintln(cmd.OutOrStdout(), \"" + name.Use + ": ok\")\n")
	b.WriteString("\t\t\treturn err\n")
	b.WriteString("\t\t},\n")
	b.WriteString("\t}\n")
	b.WriteString("}\n")
	return formatGo(b.String())
}

func renderCommandsFile(pkg string) ([]byte, error) {
	var b strings.Builder
	b.WriteString("package " + pkg + "\n\n")
	b.WriteString("import \"github.com/gombit-dev/gombit/cli\"\n\n")
	b.WriteString("// RegisterCommands attaches this feature-package's management commands to the\n")
	b.WriteString("// app-owned gombit Cobra tree. Called explicitly from cmd/gombit; Gombit does\n")
	b.WriteString("// not discover commands by reflection. Use cli.AddCommand (D13 / ADR-014).\n")
	b.WriteString("func RegisterCommands(root *cli.Command) {\n")
	b.WriteString("}\n")
	return formatGo(b.String())
}

func formatGo(src string) ([]byte, error) {
	formatted, err := format.Source([]byte(src))
	if err != nil {
		return nil, fmt.Errorf("commandgen: format source: %w", err)
	}
	return formatted, nil
}
