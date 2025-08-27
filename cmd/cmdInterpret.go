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
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

var interpretCmd = &cobra.Command{
	Use:     "interpret",
	Short:   "Generate program‐specific configs from EDN annotations",
	Long:    helpInterpret,
	Example: exampleInterpret,

	Run: runInterpret,
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func init() {
	rootCmd.AddCommand(interpretCmd)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

var fnRe = regexp.MustCompile(`^([OESRTWCQ]+)(f[0-9]+)$`)
var charRe = regexp.MustCompile(`^([OESRTWCQ]+)([a-z])$`)

var prefixMaps = map[string]map[rune]string{
	"micro": {
		'O': "Alt", 'E': "Alt",
		'T': "Ctrl", 'W': "Ctrl",
		'S': "Shift", 'R': "Shift",
	},
	"helix": {
		'O': "A", 'E': "A",
		'T': "C", 'W': "C",
		'S': "S", 'R': "S",
	},
	// TODO: add more targets here, e.g. "broot": { … }
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func runInterpret(cmd *cobra.Command, args []string) {

// TODO: update error handling & comments
	if program == "" {
		// TODO: add a flag to hit all targets
		// TODO: if "helix" iterate on all "helix" targets
		log.Fatal("please pass --target <program> (e.g. micro, helix, broot)")
	}

	allRows := gatherRows(ednFile, rootDir)
	targetFiltered := filterByProgram(allRows, program)

	rawBind := make(map[string]string, len(targetFiltered))
	for _, r := range targetFiltered {
		rawBind[r.Binding] = r.Command
	}

	formatted := formatBinds(rawBind, program)

	switch program {
	case "micro":
		// emit JSON
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		if err := enc.Encode(formatted); err != nil {
			log.Fatalf("JSON marshal error: %v", err)
		}

	case "helix-common", "helix-insert", "helix-normal", "helix-select":
		// emit TOML
		// wrap your bindings under a table if you like, e.g. [keybindings]
		w := cmd.OutOrStdout()
		for key, val := range formatted {
			// TOML strings must be quoted
			fmt.Fprintf(w, "%s = %s\n", key, val)
		}

	default:
		log.Fatalf("unsupported --target %q, expected %q or %q", program, "micro", "helix")
	}

}

////////////////////////////////////////////////////////////////////////////////////////////////////

// TODO: refactor as default flag
// helper to pick the prefixMap for a given target
func getPrefixMap(target string) map[rune]string {
	if pm, ok := prefixMaps[target]; ok {
		return pm
	}
	return prefixMaps["helix"]
}

// formatBinds converts raw keys like "OTf1" → "Alt-Ctrl-F1"
// and strips the surrounding brackets from values "[Copy]" → "Copy".
func formatBinds(raw map[string]string, target string) map[string]string {
	out := make(map[string]string, len(raw))
	pm := getPrefixMap(target)

	for k, v := range raw {
		prettyKey := k
		if m := fnRe.FindStringSubmatch(k); m != nil {
			prefixRunes, fnPart := m[1], m[2]
			var parts []string
			for _, r := range prefixRunes {
				if txt, ok := pm[r]; ok {
					parts = append(parts, txt)
				}
			}
			parts = append(parts, strings.ToUpper(fnPart))
			prettyKey = strings.Join(parts, "-")
		} else if m := charRe.FindStringSubmatch(k); m != nil {
			prefixRunes, charPart := m[1], m[2]
			var parts []string
			for _, r := range prefixRunes {
				if txt, ok := pm[r]; ok {
					parts = append(parts, txt)
				}
			}
			parts = append(parts, charPart)
			prettyKey = strings.Join(parts, "-")
		}

		var prettyVal string
		switch target {
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

////////////////////////////////////////////////////////////////////////////////////////////////////
