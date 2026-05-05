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
	"log"

	"github.com/DanielRivasMD/domovoi"
	"github.com/DanielRivasMD/horus"
	"github.com/spf13/cobra"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

var embedFlags struct {
	target string
}

////////////////////////////////////////////////////////////////////////////////////////////////////

var triggerTransforms = map[string]string{
	"up_arrow":            "up",
	"down_arrow":          "dn",
	"left_arrow":          "lf",
	"right_arrow":         "rg",
	"hyphen":              "hy",
	"equal_sign":          "eq",
	"delete_or_backspace": "db",
	"open_bracket":        "ob",
	"close_bracket":       "cb",
	"semicolon":           "sc",
	"quote":               "qu",
	"backslash":           "bl",
	"comma":               "cm",
	"period":              "pe",
	"slash":               "sl",
	"return_or_enter":     "ret",
	"spacebar":            "spc",
	"right_shift":         "kR",
	"right_option":        "kE",
	"right_command":       "kQ",
	"right_control":       "kW",
	"left_command":        "kC",
	"left_option":         "kO",
	"left_control":        "kT",
	"left_shift":          "kS",
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

// TODO: add error handler horus
func runEmbed(cmd *cobra.Command, args []string) {
	paths := resolveEDNFiles("", rootFlags.rootDir)
	allEntries, err := parseEDNFiles(paths)
	if err != nil {
		log.Fatalf("EDN parsing error: %v", err)
	}
	embedConfig(allEntries, rootFlags.program)
}

////////////////////////////////////////////////////////////////////////////////////////////////////
