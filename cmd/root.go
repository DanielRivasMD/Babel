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
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/DanielRivasMD/horus"
	"github.com/spf13/cobra"
	"github.com/ttacon/chalk"
	"olympos.io/encoding/edn"
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

var (
	verbose bool
	rootDir string
)

////////////////////////////////////////////////////////////////////////////////////////////////////

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose diagnostics")
	rootCmd.PersistentFlags().StringVarP(&rootDir, "root", "R", defaultRootDir(), "Config root (recurses .edn files)")
}

////////////////////////////////////////////////////////////////////////////////////////////////////

type Row struct {
	Action  string
	Command string
	Program string

	Trigger string
	Binding string
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func completeRenderType(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"empty", "full", "default"}, cobra.ShellCompDirectiveNoFileComp
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func defaultRootDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "" // will be caught later
	}
	return filepath.Join(home, ".saiyajin", "frag")
}

// resolveFiles returns either the single --file or all .edn under --root
func resolveFiles(file, root string) []string {
	if file != "" {
		return []string{file}
	}

	var ednFiles []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".edn") {
			ednFiles = append(ednFiles, path)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("failed to scan %s: %v", root, err)
	}
	return ednFiles
}

// extractEntry finds the next ^{…}[…] pair, returns meta & vector & new position
func extractEntry(text string, startPos int) (metaStr, vecStr string, nextPos int, ok bool) {
	// find next caret
	delta := strings.IndexRune(text[startPos:], '^')
	if delta < 0 {
		return "", "", 0, false
	}
	i := startPos + delta

	// skip whitespace, expect '{'
	j := i + 1
	for j < len(text) && unicode.IsSpace(rune(text[j])) {
		j++
	}
	if j >= len(text) || text[j] != '{' {
		return extractEntry(text, i+1)
	}

	// extract metadata map literal
	braceCount := 0
	k := j
metaLoop:
	for ; k < len(text); k++ {
		switch text[k] {
		case '{':
			braceCount++
		case '}':
			braceCount--
			if braceCount == 0 {
				k++ // include closing
				break metaLoop
			}
		}
	}
	if braceCount != 0 {
		return "", "", 0, false
	}
	metaEnd := k
	metaStr = text[j:metaEnd]

	// skip to '['
	p := metaEnd
	for p < len(text) && unicode.IsSpace(rune(text[p])) {
		p++
	}
	if p >= len(text) || text[p] != '[' {
		return extractEntry(text, metaEnd)
	}

	// extract the vector literal
	bracketCount := 0
	q := p
vecLoop:
	for ; q < len(text); q++ {
		switch text[q] {
		case '[':
			bracketCount++
		case ']':
			bracketCount--
			if bracketCount == 0 {
				q++ // include closing
				break vecLoop
			}
		}
	}
	if bracketCount != 0 {
		return "", "", 0, false
	}
	vecEnd := q
	vecStr = text[p:vecEnd]
	return metaStr, vecStr, vecEnd, true
}

// loadEDNFile reads the entire EDN file into a string
func loadEDNFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		horus.CheckErr(
			err,
			horus.WithMessage(path),
			horus.WithExitCode(2),
			horus.WithFormatter(func(he *horus.Herror) string {
				return "failed to read: " + chalk.Red.Color(he.Message)
			}),
		)
	}
	return string(data)
}

// decodeMetadata turns the EDN map string into map
func decodeMetadata(metaStr string) (map[edn.Keyword]any, error) {
	var rawMeta map[edn.Keyword]any
	err := edn.Unmarshal([]byte(metaStr), &rawMeta)
	return rawMeta, err
}

// decodeRule parses the EDN vector into []any
func decodeRule(vecStr string) ([]any, error) {
	var raw any
	dec := edn.NewDecoder(strings.NewReader(vecStr))
	if err := dec.Decode(&raw); err != nil {
		return nil, err
	}
	vec, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("invalid rule form")
	}
	return vec, nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////
