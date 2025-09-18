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
	"log"
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

// TODO: upgrade flag checking
func runInterpret(cmd *cobra.Command, args []string) {
	// Resolve EDN file paths
	paths := resolveEDNFiles(flags.ednFile, flags.rootDir)

	// Parse all EDN files into structured bindings
	allEntries, err := parseEDNFiles(paths)
	if err != nil {
		log.Fatalf("EDN parsing error: %v", err)
	}

	// TODO: use normalize program?
	// Emit for multiple Helix modes
	if flags.program == "helix" {
		for _, sub := range []string{"helix-common", "helix-insert", "helix-normal", "helix-select"} {
			emitConfig(cmd, allEntries, sub)
			fmt.Fprintln(cmd.OutOrStdout())
		}
		return
	}

	// Emit for single target
	emitConfig(cmd, allEntries, flags.program)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func emitConfig(cmd *cobra.Command, entries []BindingEntry, target string) {
	filtered := filterByProgram(entries, target)

	rawBind := make(map[string]string)
	for _, entry := range filtered {
		for _, act := range entry.Actions {
			bindKey := formatKeySeq(entry.Binding, lookups.config, act.Program)
			rawBind[bindKey] = act.Command
		}
	}

	formatted := formatBinds(rawBind, target)
	w := cmd.OutOrStdout()

	switch target {
	case "helix-common", "helix-insert", "helix-normal", "helix-select":
		if headerLines, ok := programHeaders[target]; ok {
			for _, line := range headerLines {
				fmt.Fprintln(w, line)
			}
		}
		for key, val := range formatted {
			fmt.Fprintf(w, "%s = %s\n", key, val)
		}

	case "micro":
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

// Format values for TOML output
func formatBinds(raw map[string]string, program string) map[string]string {
	out := make(map[string]string, len(raw))

	for k, v := range raw {
		var prettyVal string
		switch program {
		case "helix-common", "helix-insert", "helix-normal", "helix-select":
			prettyVal = tomlList(v)
		case "micro":
			prettyVal = strings.Trim(v, "[]")
		default:
			prettyVal = v
		}
		out[k] = prettyVal
	}
	return out
}

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

var programHeaders = map[string][]string{
	"helix-common": {},
	"helix-insert": {
		"[keys.insert]",
		`A-ret = ["completion"]`,
		`esc = ["normal_mode"]`},
	"helix-normal": {
		"[keys.normal]",
		`esc = ["collapse_selection", "keep_primary_selection"]`,
		`ret = ["insert_mode"]`,
		`A-ret = ["hover"]`},
	"helix-select": {

		"[keys.select]",
		`esc = ["normal_mode"]`,
		`ret = ["insert_mode"]`,
		`A-ret = ["hover"]`,
	},
	"micro": {
		`"MouseRight": "MouseMultiCursor",`,
		`"AltEnter": "Autocomplete",`,
	},
}

////////////////////////////////////////////////////////////////////////////////////////////////////
