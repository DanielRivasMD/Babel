/*
Copyright © 2025 Daniel Rivas <danielrivasmd@gmail.com>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/
package cmd

////////////////////////////////////////////////////////////////////////////////////////////////////

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/DanielRivasMD/horus"
	"github.com/spf13/cobra"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

var interpretCmd = &cobra.Command{
	Use:     "interpret",
	Short:   "Generate program‐specific configs from EDN annotations",
	Long:    helpInterpret,
	Example: exampleInterpret,

	PreRun: preInterpret,
	Run:    runInterpret,
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func init() {
	rootCmd.AddCommand(interpretCmd)

	interpretCmd.Flags().StringVarP(&flags.interpretTarget, "target", "t", "", "Write output to this file instead of stdout")
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func preInterpret(cmd *cobra.Command, args []string) {
	horus.CheckEmpty(
		flags.program,
		"",
		horus.WithMessage("`--program` is required"),
		horus.WithExitCode(2),
		horus.WithFormatter(func(he *horus.Herror) string { return onelineErr(he.Message) }),
	)
	horus.CheckEmpty(
		flags.rootDir,
		"",
		horus.WithMessage("`--root` is required"),
		horus.WithExitCode(2),
		horus.WithFormatter(func(he *horus.Herror) string { return onelineErr(he.Message) }),
	)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func runInterpret(cmd *cobra.Command, args []string) {
	// Resolve EDN file paths
	paths := resolveEDNFiles(flags.ednFile, flags.rootDir)

	// Parse all EDN files into structured bindings
	allEntries, err := parseEDNFiles(paths)
	if err != nil {
		log.Fatalf("EDN parsing error: %v", err)
	}

	// Decide output writer
	var w io.Writer = cmd.OutOrStdout()
	if flags.interpretTarget != "" {
		f, err := os.Create(flags.interpretTarget)
		if err != nil {
			log.Fatalf("failed to create target file %q: %v", flags.interpretTarget, err)
		}
		defer f.Close()
		w = f
	}

	// Define program families (expand only on exact family names)
	families := map[string][]string{
		"helix": {"helix-common", "helix-insert", "helix-normal", "helix-select"},
		"micro": {"micro"},
	}

	// respect exact targets
	if bases, ok := families[flags.program]; ok {
		for _, b := range bases {
			emitConfig(w, allEntries, b)
			fmt.Fprintln(w)
		}
		return
	}

	// Default: emit once for the exact program provided
	emitConfig(w, allEntries, flags.program)
	fmt.Fprintln(w)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func emitConfig(w io.Writer, entries []BindingEntry, target string) {
	filtered := filterByProgram(entries, target)

	rawBind := make(map[string]string)
	for _, entry := range filtered {
		for _, actions := range entry.Actions {
			bindKey := formatKeySeq(entry.Binding, lookups.interpret, actions.Program, "-")
			rawBind[bindKey] = actions.Command
		}
	}

	formatted := formatBinds(rawBind, target)

	switch {
	case strings.HasPrefix(target, "helix-"):
		if headerLines, ok := programHeaders[target]; ok {
			for _, line := range headerLines {
				fmt.Fprintln(w, line)
			}
		}
		for key, val := range formatted {
			fmt.Fprintf(w, "%s = %s\n", key, val)
		}

	case target == "micro":
		fmt.Fprintln(w, "{")
		if headerLines, ok := programHeaders[target]; ok {
			for _, line := range headerLines {
				fmt.Fprintln(w, line)
			}
		}
		for key, val := range formatted {
			fmt.Fprintf(w, "  %q: %q,\n", key, val)
		}
		fmt.Fprintln(w, "}")

	default:
		log.Fatalf("unsupported --program %q", target)
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func formatBinds(raw map[string]string, program string) map[string]string {
	out := make(map[string]string, len(raw))

	for k, v := range raw {
		var prettyVal string
		switch {
		case strings.HasPrefix(program, "helix-"):
			prettyVal = tomlList(v)

		case program == "micro",
			program == "lazygit",
			program == "zellij":
			prettyVal = strings.Trim(v, "[]")

		default:
			prettyVal = v
		}
		out[k] = prettyVal
	}
	return out
}

////////////////////////////////////////////////////////////////////////////////////////////////////

// Convert EDN-style list to TOML array
func tomlList(raw string) string {
	inner := strings.TrimSpace(raw)
	inner = strings.TrimPrefix(inner, "[")
	inner = strings.TrimSuffix(inner, "]")

	if strings.HasPrefix(inner, ":sh ") {
		return fmt.Sprintf("[%q]", inner)
	}

	if inner == "" {
		return "[]"
	}

	parts := strings.Fields(inner)
	quoted := make([]string, len(parts))
	for i, p := range parts {
		quoted[i] = fmt.Sprintf("%q", p)
	}
	return "[" + strings.Join(quoted, ",") + "]"
}

////////////////////////////////////////////////////////////////////////////////////////////////////

// TODO: pass as config toml
var programHeaders = map[string][]string{
	"helix-common": {},
	"helix-insert": {
		"[keys.insert]",
		`A-ret = ["completion"]`,
	},
	"helix-normal": {
		"[keys.normal]",
		`A-ret = ["hover"]`,
	},
	"helix-select": {
		"[keys.select]",
		`A-ret = ["hover"]`,
	},
	"micro": {
		`"MouseRight": "MouseMultiCursor",`,
		`"AltEnter": "Autocomplete",`,
	},
}

////////////////////////////////////////////////////////////////////////////////////////////////////
