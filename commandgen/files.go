package commandgen

import (
	"go/format"
	"strings"
)

type fileSpec struct {
	relPath string
	content []byte
	owned   bool
}

func renderCommandFile(name CommandName) []byte {
	var b strings.Builder
	b.WriteString("// " + GeneratedBanner + "\n")
	b.WriteString("package " + name.Package + "\n\n")
	b.WriteString("import (\n")
	b.WriteString("\t\"fmt\"\n\n")
	b.WriteString("\t\"github.com/LAA-Software-Engineering/gombit/cli\"\n")
	b.WriteString(")\n\n")
	b.WriteString("// " + name.Constructor + " returns the " + name.Use + " management command.\n")
	b.WriteString("// Attach it with cli.AddCommand from Register (D13 / ADR-014).\n")
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
	return mustFormatGo(b.String())
}

func renderRegisterFile(pkg string) []byte {
	var b strings.Builder
	b.WriteString("package " + pkg + "\n\n")
	b.WriteString("import \"github.com/LAA-Software-Engineering/gombit/cli\"\n\n")
	b.WriteString("// Register attaches this feature-package's management commands to the\n")
	b.WriteString("// app-owned gombit Cobra tree via cli.AddCommand (D13 / ADR-014).\n")
	b.WriteString("// Called explicitly from cmd/gombit; Gombit does not discover commands\n")
	b.WriteString("// by reflection.\n")
	b.WriteString("func Register(root *cli.Command) {\n")
	b.WriteString("}\n")
	return mustFormatGo(b.String())
}

func mustFormatGo(src string) []byte {
	formatted, err := format.Source([]byte(src))
	if err != nil {
		return []byte(src + "\n// format error: " + err.Error() + "\n")
	}
	return formatted
}
