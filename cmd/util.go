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
	"path/filepath"
	"regexp"
	"strings"

	"github.com/DanielRivasMD/domovoi"
	"github.com/DanielRivasMD/horus"
	"github.com/spf13/cobra"
	"github.com/ttacon/chalk"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

func defaultRootDir() string {
	home, err := domovoi.FindHome(false)
	horus.CheckErr(err, horus.WithCategory("init_error"), horus.WithMessage("getting home directory"))
	return filepath.Join(home, ".saiyajin", "edn")
}

////////////////////////////////////////////////////////////////////////////////////////////////////

// TODO: load completions from toml or const
func completeRenderType(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"empty", "full", "default"}, cobra.ShellCompDirectiveNoFileComp
}

// TODO: add static sorting for display rendering instead of alphabetical
func completeSortType(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	options := []string{"program", "action", "trigger", "binding"}
	var completions []string
	for _, opt := range options {
		if strings.HasPrefix(opt, toComplete) {
			completions = append(completions, opt)
		}
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}

func completePrograms(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"helix", "helix-common", "helix-insert", "helix-normal", "helix-select", "kanata", "micro", "serpl", "zellij"}, cobra.ShellCompDirectiveNoFileComp
}

////////////////////////////////////////////////////////////////////////////////////////////////////

var (
	rg = map[string]*regexp.Regexp{
		"fn": regexp.MustCompile(`^([OESRTWCQ]+)(f[0-9]+)$`),
		"ch": regexp.MustCompile(`^([OESRTWCQ]+)([a-z])$`),
		"nb": regexp.MustCompile(`^([OESRTWCQ]+)([0-9])$`),
		"ot": regexp.MustCompile(`^([OESRTWCQ]+)([a-z_]+)$`),
		"kw": regexp.MustCompile(`^([OESRTWCQ]*)#P(.+)$`),
	}

	lookups lookUps

	programColors = map[string]chalk.Color{
		"micro":        chalk.Cyan,
		"helix-common": chalk.Cyan,
		"helix-insert": chalk.Cyan,
		"helix-normal": chalk.Cyan,
		"helix-pop":    chalk.Cyan,
		"helix-select": chalk.Cyan,
		"lazygit":      chalk.Green,
		"serpl":        chalk.Green,
		"terminal":     chalk.Blue,
		"zellij":       chalk.Yellow,
	}

	tableBorder  = "=================================================================================================="
	tableHeader  = "| Program         | Action                         | Trigger              | Binding              |"
	tableDivider = "|-----------------|--------------------------------|----------------------|----------------------|"

	TC = "TC" // prefix used in EDN parsing for markdown generation

	// TODO: move away from hardcoded variables => load from template
	programHeaders = map[string][]string{
		"helix-common": {},
		"helix-insert": {
			"[keys.insert]",
			`A-ret = ["completion"]`,
		},
		"helix-normal": {
			"[keys.normal]",
			`A-ret = ["hover"]`,
			// `"`" = "no_op"`,
			// `"A-`" = "no_op"`,
			`"," = "repeat_last_motion"`,
			`g = "no_op"`,
			`G = "no_op"`,
			`Z = "no_op"`,
			`"~" = "no_op"	`,
			`"=" = "no_op"`,
			`"<" = "no_op"`,
			`">" = "no_op"`,
			`q = "no_op"`,
			`Q = "no_op"`,
			`"|" = "no_op"`,
			`"A-|" = "no_op"`,
			`"!" = "no_op"`,
			`"A-!" = "no_op"`,
			`"$" = "no_op"`,
			`S = "no_op"`,
			`"A-_" = "no_op"`,
			`"&" = "no_op"`,
			`"_" = "no_op"`,
			`"A-;" = "no_op"`,
			`"A-:" = "no_op"`,
			// `"," = "no_op"`,
			`"A-," = "no_op"`,
			`C = "no_op"`,
			`"(" = "no_op"`,
			`")" = "no_op"`,
			`"A-(" = "no_op"`,
			`"A-)" = "no_op"`,
			`"%" = "no_op"`,
			`x = "no_op"`,
			`X = "no_op"`,
			`J = "no_op"`,
			`K = "no_op"`,
			`"C-c" = "no_op"`,
			// `"/" = "no_op"`,
			`"?" = "no_op"`,
			`m = "no_op"`,
			`n = "no_op"`,
			`N = "no_op"`,
			`"A-*" = "no_op"`,
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
)

////////////////////////////////////////////////////////////////////////////////////////////////////
