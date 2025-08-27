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

var (
	target string
)

////////////////////////////////////////////////////////////////////////////////////////////////////

func init() {
	rootCmd.AddCommand(interpretCmd)

	interpretCmd.Flags().StringVarP(&target, "target", "t", "", "Which program to emit (e.g. micro, helix, broot)")
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func runInterpret(cmd *cobra.Command, args []string) {
	// 0) require flags
	if target == "" {
		log.Fatal("please pass --target <program> (e.g. micro, helix, broot)")
	}
	if ednFile == "" && rootDir == "" {
		log.Fatal("please pass --file <path>.edn or --root <config-dir>")
	}

	// 1) collect all EDN rows into []Row
	files := resolveFiles(ednFile, rootDir)
	var allRows []Row
	for _, path := range files {
		text := loadEDNFile(path)
		mode := extractMode(text)
		allRows = append(allRows, parseBindings(text, mode)...)
	}

	// 2) filter for our target program
	var rows []Row
	for _, r := range allRows {
		if r.Program == target {
			rows = append(rows, r)
		}
	}

	// 3) build raw key→command map
	rawBind := make(map[string]string, len(rows))
	for _, r := range rows {
		rawBind[r.Binding] = r.Command
	}

	// 3a) now pretty‐print every key via formatBinds
	formatted := formatBinds(rawBind)
	fmt.Println(rawBind)
	fmt.Println(formatted)

	// 4) emit JSON of the pretty map
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	if err := enc.Encode(formatted); err != nil {
		log.Fatalf("JSON marshal error: %v", err)
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////

var prefixMap = map[rune]string{
	'O': "Alt", 'E': "Alt",
	'T': "Ctrl", 'Q': "Ctrl",
	'S': "Shift", 'R': "Shift",
}

var fnRe = regexp.MustCompile(`^([OESTQR]+)(f[0-9]+)$`)

// formatBinds converts raw keys like "OTf1" → "Alt-Ctrl-F1"
// and strips the surrounding brackets from values "[Copy]" → "Copy".
func formatBinds(raw map[string]string) map[string]string {
	out := make(map[string]string, len(raw))

	for k, v := range raw {
		prettyKey := k
		// 1) detect "prefixes"+"f<digits>"
		if m := fnRe.FindStringSubmatch(k); m != nil {
			prefixRunes, fnPart := m[1], m[2] // e.g. "OT", "f1"
			var parts []string
			for _, r := range prefixRunes {
				if txt, ok := prefixMap[r]; ok {
					parts = append(parts, txt)
				}
			}
			// uppercase F<digits>
			fnPart = strings.ToUpper(fnPart)
			parts = append(parts, fnPart)
			prettyKey = strings.Join(parts, "-")
		}

		// 2) strip leading/trailing brackets from the command string
		prettyVal := strings.TrimPrefix(v, "[")
		prettyVal = strings.TrimSuffix(prettyVal, "]")

		out[prettyKey] = prettyVal
	}

	return out
}

////////////////////////////////////////////////////////////////////////////////////////////////////
