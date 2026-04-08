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

	"github.com/DanielRivasMD/domovoi"
	"github.com/DanielRivasMD/horus"
	"github.com/spf13/cobra"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

var embedFlags struct {
	target string
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func EmbedCmd() *cobra.Command {
	d := horus.Must(domovoi.GlobalDocs())
	cmd := horus.Must(d.MakeCmd("embed", runEmbed))

	cmd.Flags().StringVarP(&embedFlags.target, "target", "", "", "Config file to supplement")
	cmd.PreRun = preEmbed

	return cmd
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func preEmbed(cmd *cobra.Command, args []string) {
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

func runEmbed(cmd *cobra.Command, args []string) {
	paths := resolveEDNFiles("", rootFlags.rootDir)
	allEntries, err := parseEDNFiles(paths)
	if err != nil {
		log.Fatalf("EDN parsing error: %v", err)
	}
	embedConfig(allEntries, rootFlags.program)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func embedConfig(entries []BindingEntry, target string) {
	filtered := filterByProgram(entries, target)

	switch {
	case target == "kanata":
	case target == "serpl":
	case target == "lazygit":
		rawBind := make(map[string]string)
		for _, entry := range filtered {
			for _, act := range entry.Actions {
				bindKey := formatKeySeq(entry.Binding, lookups.embed, act.Program, "-")
				rawBind[bindKey] = act.Command
			}
		}
		formatted := formatBinds(rawBind, target)
		replaces := []moldReplace{}
		for key, val := range formatted {
			replaces = append(replaces,
				replace(val, fmt.Sprintf("    %s: '<%s>':line", val, key)))
		}
		mf := newMoldConfig(embedFlags.target, []string{embedFlags.target}, replaces...)
		moldForging("embed-lazygit", mf)

	case strings.HasPrefix(target, "zellij"):
		normalized := normalizeProgram(target)
		replaces := []moldReplace{}
		for _, entry := range filtered {
			for _, act := range entry.Actions {
				bindKey := formatKeySeq(entry.Binding, lookups.embed, normalized, " ")
				replaces = append(replaces, formatZellijReplace(bindKey, act))
			}
		}
		moldForging(
			"embed-zellij",
			newMoldConfig(embedFlags.target, []string{embedFlags.target}, replaces...),
		)

	default:
		log.Fatalf("unsupported --program %q", target)
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func formatZellijReplace(key string, act ProgramAction) moldReplace {
	escapedCmd := escapeForMold(act.Command)
	escapedCmd = strings.Trim(escapedCmd, "[]")
	lhs := escapedCmd
	rhs := fmt.Sprintf("        bind \\\"%s\\\" { %s }:line", key, escapedCmd)
	return replace(fmt.Sprintf("\"%s\"", lhs), rhs)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func escapeForMold(cmd string) string {
	return strings.ReplaceAll(cmd, `"`, `\"`)
}

////////////////////////////////////////////////////////////////////////////////////////////////////
