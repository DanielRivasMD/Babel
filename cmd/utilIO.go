////////////////////////////////////////////////////////////////////////////////////////////////////

package cmd

////////////////////////////////////////////////////////////////////////////////////////////////////

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/DanielRivasMD/horus"
	"github.com/ttacon/chalk"
	"olympos.io/encoding/edn"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

// TODO: update default root dir definition
func defaultRootDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "" // will be caught later
	}
	return filepath.Join(home, ".saiyajin", "frag")
}

// TODO: update error habdling
// resolveEDNFiles returns either the single --file or all .edn under --root
func resolveEDNFiles(file, root string) []string {
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
