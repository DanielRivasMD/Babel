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

	"github.com/DanielRivasMD/domovoi"
	"github.com/DanielRivasMD/horus"
	"github.com/spf13/cobra"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

var interpretFlags struct {
	target string
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func InterpretCmd() *cobra.Command {
	d := horus.Must(domovoi.GlobalDocs())
	cmd := horus.Must(d.MakeCmd("interpret", runInterpret))

	cmd.Flags().StringVarP(&interpretFlags.target, "target", "t", "", "Write output to this file instead of stdout")
	cmd.PreRun = preInterpret

	return cmd
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func preInterpret(cmd *cobra.Command, args []string) {
	horus.CheckEmpty(rootFlags.program, "",
		horus.WithMessage("`--program` is required"),
		horus.WithExitCode(2),
		horus.WithFormatter(func(he *horus.Herror) string { return horus.OneLineErr(he.Message) }))
	horus.CheckEmpty(rootFlags.rootDir, "",
		horus.WithMessage("`--root` is required"),
		horus.WithExitCode(2),
		horus.WithFormatter(func(he *horus.Herror) string { return horus.OneLineErr(he.Message) }))
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func runInterpret(cmd *cobra.Command, args []string) {
	paths := resolveEDNFiles("", rootFlags.rootDir)
	allEntries, err := parseEDNFiles(paths)
	if err != nil {
		log.Fatalf("EDN parsing error: %v", err)
	}

	var w io.Writer = cmd.OutOrStdout()
	if interpretFlags.target != "" {
		f, err := os.Create(interpretFlags.target)
		if err != nil {
			log.Fatalf("failed to create target file %q: %v", interpretFlags.target, err)
		}
		defer f.Close()
		w = f
	}

	families := map[string][]string{
		"helix": {"helix-common", "helix-insert", "helix-normal", "helix-select"},
		"micro": {"micro"},
	}
	if bases, ok := families[rootFlags.program]; ok {
		for _, b := range bases {
			emitConfig(w, allEntries, b)
			fmt.Fprintln(w)
		}
		return
	}
	emitConfig(w, allEntries, rootFlags.program)
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
