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

	"github.com/DanielRivasMD/domovoi"
	"github.com/DanielRivasMD/horus"
	"github.com/spf13/cobra"
	"github.com/ttacon/chalk"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

var rootCmd = &cobra.Command{
	Use:     "babel",
	Long:    helpRoot,
	Example: exampleRoot,
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func Execute() {
	horus.CheckErr(rootCmd.Execute())
}

////////////////////////////////////////////////////////////////////////////////////////////////////

var rg = map[string]*regexp.Regexp{
	"fn": regexp.MustCompile(`^([OESRTWCQ]+)(f[0-9]+)$`),
	"ch": regexp.MustCompile(`^([OESRTWCQ]+)([a-z])$`),
	"nb": regexp.MustCompile(`^([OESRTWCQ]+)([0-9])$`),
	"ot": regexp.MustCompile(`^([OESRTWCQ]+)([a-z_]+)$`),
	"kw": regexp.MustCompile(`^([OESRTWCQ]*)#P(.+)$`), // fallback for keywords like "!O#Ppage_up"
}

var (
	dirs    configDirs
	flags   babelFlags
	lookups lookUps
)

type configDirs struct {
	home   string
	babel  string
	config string
}

type babelFlags struct {
	// root
	verbose bool
	rootDir string
	program string

	// display
	ednFile    string
	renderMode string
	sortBy     string

	// embed
	embedTarget string
}

type lookUps struct {
	displayBinding map[string]KeyLookup
	displayTrigger map[string]KeyLookup
	interpret      map[string]KeyLookup
	embed          map[string]KeyLookup
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func init() {
	rootCmd.PersistentFlags().BoolVarP(&flags.verbose, "verbose", "v", false, "Enable verbose diagnostics")
	rootCmd.PersistentFlags().StringVarP(&flags.program, "program", "", "", "Regex or substring to filter Program names (e.g. helix)")
	rootCmd.PersistentFlags().StringVarP(&flags.rootDir, "root", "", defaultRootDir(), "Config root (recurses .edn files)")

	horus.CheckErr(
		displayCmd.RegisterFlagCompletionFunc("program", completePrograms),
		horus.WithOp("root.init"),
		horus.WithMessage("registering config completion for flag program"),
	)

	cobra.OnInitialize(initConfigDirs)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func initConfigDirs() {
	var err error
	dirs.home, err = domovoi.FindHome(flags.verbose)
	horus.CheckErr(err, horus.WithCategory("init_error"), horus.WithMessage("getting home directory"))
	dirs.babel = filepath.Join(dirs.home, ".babel")
	dirs.config = filepath.Join(dirs.babel, "config")

	lookups.displayBinding = buildLookupFuncs(loadFormat(filepath.Join(dirs.config, "display_binding.toml")))
	lookups.displayTrigger = buildLookupFuncs(loadFormat(filepath.Join(dirs.config, "display_trigger.toml")))
	lookups.interpret = buildLookupFuncs(loadFormat(filepath.Join(dirs.config, "interpret.toml")))
	lookups.embed = buildLookupFuncs(loadFormat(filepath.Join(dirs.config, "embed.toml")))
}

func onelineErr(er string) string {
	return chalk.Bold.TextStyle(chalk.Red.Color(er))
}

////////////////////////////////////////////////////////////////////////////////////////////////////

// TODO: add config for binding interpret & display
type KeySeq struct {
	Mode     string
	Modifier string
	Key      string
}

type ProgramAction struct {
	Program string
	Action  string
	Command string
}

type BindingEntry struct {
	Trigger     KeySeq
	Binding     KeySeq
	Sequence    string
	Actions     []ProgramAction
	Annotations map[string][]string // e.g. "alone" -> ["f13"]
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func completeRenderType(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"empty", "full", "default"}, cobra.ShellCompDirectiveNoFileComp
}

func completePrograms(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"helix", "helix-common", "helix-insert", "helix-normal", "helix-select", "micro"}, cobra.ShellCompDirectiveNoFileComp
}

////////////////////////////////////////////////////////////////////////////////////////////////////
