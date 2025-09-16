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
	"encoding/json"
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
	// Validate flags (should be done in preInterpret)
	if flags.program == "" {
		log.Fatalf("missing --program")
	}
	if flags.rootDir == "" && flags.ednFile == "" {
		log.Fatalf("missing --root or --file")
	}

	// Resolve EDN file paths
	paths := resolveEDNFiles(flags.ednFile, flags.rootDir)

	// Parse all EDN files into structured bindings
	allEntries, err := parseEDNFiles(paths)
	if err != nil {
		log.Fatalf("EDN parsing error: %v", err)
	}

	// fmt.Println("ALL ENTRIES")
	// for _, e := range allEntries {
	// 	fmt.Println(e)
	// }
	// fmt.Println(allEntries)

	// Emit for multiple Helix modes
	if flags.program == "helix" {
		for _, sub := range []string{"helix-common", "helix-insert", "helix-normal", "helix-select"} {
			emitBindings(cmd, allEntries, sub)
			fmt.Fprintln(cmd.OutOrStdout())
		}
		return
	}

	// Emit for single target
	emitBindings(cmd, allEntries, flags.program)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

// formatBinds converts raw keys like "OTf1" → "Alt-Ctrl-F1"
// and strips the surrounding brackets from values "[Copy]" → "Copy".
func formatBinds(raw map[string]string, program string) map[string]string {
	out := make(map[string]string, len(raw))
	// pm := getPrefixMap(program)

	for k, v := range raw {
		key := stripEDNPrefix(k)
		prettyKey := key

		// if m := rg["fn"].FindStringSubmatch(key); m != nil {
		// 	prefixRunes, fnPart := m[1], m[2]
		// 	var parts []string
		// 	for _, r := range prefixRunes {
		// 		if txt, ok := pm[r]; ok {
		// 			parts = append(parts, txt)
		// 		}
		// 	}
		// 	km := getKeyMap(program)
		// 	if mapped, ok := km[fnPart]; ok {
		// 		parts = append(parts, mapped)
		// 	} else {
		// 		parts = append(parts, strings.ToUpper(fnPart))
		// 	}

		// 	// parts = append(parts, strings.ToUpper(fnPart))
		// 	prettyKey = strings.Join(parts, "-")

		// } else if m := rg["ch"].FindStringSubmatch(key); m != nil {
		// 	prefixRunes, charPart := m[1], m[2]
		// 	var parts []string
		// 	for _, r := range prefixRunes {
		// 		if txt, ok := pm[r]; ok {
		// 			parts = append(parts, txt)
		// 		}
		// 	}
		// 	parts = append(parts, charPart)
		// 	prettyKey = strings.Join(parts, "-")
		// }

		var prettyVal string
		switch program {
		case "micro":
			prettyVal = strings.Trim(v, "[]")
		case "helix-common", "helix-insert", "helix-normal", "helix-select":
			prettyVal = tomlList(v)
		}
		out[prettyKey] = prettyVal
	}
	return out
}

// tomlList converts a bracketed space‐separated string
// into a quoted, comma‐separated TOML array.
// e.g. "[a b c]" → ["a","b","c"].
func tomlList(raw string) string {
	// strip the outer brackets and any whitespace
	inner := strings.TrimSpace(raw)
	inner = strings.TrimPrefix(inner, "[")
	inner = strings.TrimSuffix(inner, "]")

	// exception: shell commands
	if strings.HasPrefix(inner, ":sh ") {
		// emit a single quoted string
		return fmt.Sprintf("[%q]", inner)
	}

	if inner == "" {
		return "[]"
	}

	// split on whitespace
	parts := strings.Fields(inner)

	// quote each element
	quoted := make([]string, len(parts))
	for i, p := range parts {
		quoted[i] = fmt.Sprintf("%q", p)
	}

	// join into a TOML array
	return "[" + strings.Join(quoted, ",") + "]"
}

func emitBindings(cmd *cobra.Command, entries []BindingEntry, target string) {
	// Filter actions by target program
	filtered := filterByProgram(entries, target)

	// Build raw binding map: key → command
	rawBind := make(map[string]string)
	for _, entry := range filtered {
		for _, act := range entry.Actions {
			// Use modifier + key as binding identifier
			bindKey := entry.Binding.Modifier + "-" + strings.ToUpper(entry.Binding.Key)
			rawBind[bindKey] = act.Command
		}
	}

	// fmt.Println(rawBind)

	// Format bindings for output
	formatted := formatBinds(rawBind, target)

	// Emit based on target format
	switch target {
	case "micro":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		if err := enc.Encode(formatted); err != nil {
			log.Fatalf("JSON marshal error: %v", err)
		}

	case "helix-common", "helix-insert", "helix-normal", "helix-select":
		w := cmd.OutOrStdout()
		fmt.Fprintf(w, "# mode: %s\n", target)
		for key, val := range formatted {
			fmt.Fprintf(w, "%s = %s\n", key, val)
		}

	default:
		log.Fatalf("unsupported --program %q", target)
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////
