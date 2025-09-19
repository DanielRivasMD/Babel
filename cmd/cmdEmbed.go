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

var embedCmd = &cobra.Command{
	Use:     "embed",
	Short:   "",
	Long:    helpEmbed,
	Example: exampleEmbed,

	PreRun: preEmbed,
	Run:    runEmbed,
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func init() {
	rootCmd.AddCommand(embedCmd)

	embedCmd.Flags().StringVarP(&flags.embedTarget, "target", "", "", "Config file to supplement")
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func preEmbed(cmd *cobra.Command, args []string) {
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

func runEmbed(cmd *cobra.Command, args []string) {
	// Resolve EDN file paths
	paths := resolveEDNFiles(flags.ednFile, flags.rootDir)

	// Parse all EDN files into structured bindings
	allEntries, err := parseEDNFiles(paths)
	if err != nil {
		log.Fatalf("EDN parsing error: %v", err)
	}

	// Embed for single target
	embedConfig(allEntries, flags.program)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func embedConfig(entries []BindingEntry, target string) {
	filtered := filterByProgram(entries, target)

	rawBind := make(map[string]string)
	for _, entry := range filtered {
		for _, actions := range entry.Actions {
			bindKey := formatKeySeq(entry.Binding, lookups.embed, actions.Program)
			rawBind[bindKey] = actions.Command
		}
	}

	formatted := formatBinds(rawBind, target)

	switch target {
	case "broot":

	case "lazygit":
		// collect replacements from edn metadata
		replaces := []mbomboReplace{}
		for key, val := range formatted {
			// TODO: relocate trimming to a function
			clean := strings.Trim(val, "[]")
			// DOC: all config values in lazygit are tabbed 4 spaces
			replaces = append(replaces, Replace(clean, fmt.Sprintf("    %s: '<%s>':line", clean, key)))
		}

		mf := newMbomboConfig(
			flags.embedTarget,
			[]string{flags.embedTarget},
			replaces...,
		)

		mbomboForging("embed-lazygit", mf)

	default:
		log.Fatalf("unsupported --program %q", target)
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////
