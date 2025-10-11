////////////////////////////////////////////////////////////////////////////////////////////////////

package cmd

////////////////////////////////////////////////////////////////////////////////////////////////////

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/DanielRivasMD/domovoi"
	"github.com/DanielRivasMD/horus"
	"github.com/ttacon/chalk"
	"olympos.io/encoding/edn"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

// TODO: load values on config
func formatKeySeq(k KeySeq, lookups map[string]KeyLookup, program, sep string) string {
	lookup := lookups[normalizeProgram(program)]
	if lookup == nil {
		lookup = lookups["default"]
	}

	var modParts []string
	for _, r := range k.Modifier {
		modParts = append(modParts, lookup(string(r)))
	}
	mod := strings.Join(modParts, sep)

	// Capitalize function keys like f1 → F1
	key := normalizeFunctionKey(k.Key)
	mapped := lookup(key)

	var out string
	if mod != "" {
		out = mod + sep + mapped
	} else {
		out = mapped
	}

	if k.Mode != "" {
		return "(" + k.Mode + ")" + " " + out
	}
	return out
}

func formatBindingEntry(b BindingEntry, lookups map[string]KeyLookup, program string) string {
	lookup := lookups[normalizeProgram(program)]
	if lookup == nil {
		lookup = lookups["default"]
	}

	key := b.Sequence
	if key == "" {
		key = b.Binding.Key
	}

	// Capitalize function keys like f1 → F1
	key = normalizeFunctionKey(key)

	var modParts []string
	for _, r := range b.Binding.Modifier {
		modParts = append(modParts, lookup(string(r)))
	}
	mod := strings.Join(modParts, "-")

	var out string
	if mod != "" {
		out = mod + "-" + lookup(key)
	} else {
		out = lookup(key)
	}
	return out
}

func loadFormat(path string) map[string]map[string]string {
	var cfg map[string]map[string]string
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		log.Fatalf("failed to load format config from %s: %v", path, err)
	}
	return cfg
}

type KeyLookup func(string) string

func buildLookupFuncs(cfg map[string]map[string]string) map[string]KeyLookup {
	defaultMap := cfg["default"]
	out := make(map[string]KeyLookup)

	for program, mapping := range cfg {
		local := mapping // capture per iteration

		out[program] = func(local map[string]string) KeyLookup {
			return func(key string) string {
				if val, ok := local[key]; ok {
					return val
				}
				if val, ok := defaultMap[key]; ok {
					return val
				}
				return key
			}
		}(local) // pass explicitly
	}

	out["default"] = func(key string) string {
		if val, ok := defaultMap[key]; ok {
			return val
		}
		return key
	}

	return out
}

func normalizeProgram(p string) string {
	// Collapse all zellij-* variants to "zellij"
	if strings.HasPrefix(p, "zellij-") {
		return "zellij"
	}

	switch p {
	case "helix-common", "helix-insert", "helix-normal", "helix-select",
		"macosx-helix-common", "macosx-helix-insert", "macosx-helix-normal", "macosx-helix-select",
		"ubuntu-helix-common", "ubuntu-helix-insert", "ubuntu-helix-normal", "ubuntu-helix-select":
		return "helix"
	default:
		return p
	}
}

func normalizeFunctionKey(key string) string {
	if strings.HasPrefix(strings.ToLower(key), "f") && len(key) > 1 {
		if _, err := strconv.Atoi(key[1:]); err == nil {
			return "F" + key[1:]
		}
	}
	return key
}

////////////////////////////////////////////////////////////////////////////////////////////////////

// TODO: update default root dir definition
func defaultRootDir() string {
	home, err := domovoi.FindHome(false)
	horus.CheckErr(err, horus.WithCategory("init_error"), horus.WithMessage("getting home directory"))
	return filepath.Join(home, ".saiyajin", "edn")
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
