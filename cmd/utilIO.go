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

	"github.com/BurntSushi/toml"
	"github.com/DanielRivasMD/horus"
	"github.com/ttacon/chalk"
	"olympos.io/encoding/edn"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

var bindingLookups = buildLookupFuncs(loadBindingFormat("binding.toml"))
var triggerLookups = buildLookupFuncs(loadTriggerFormat("trigger.toml"))

func normalizeProgram(p string) string {
	switch p {
	case "helix-common", "helix-insert", "helix-normal", "helix-select":
		return "helix"
	default:
		return p
	}
}

func formatTrigger(k KeySeq, program string) string {
	lookup := triggerLookups[normalizeProgram(program)]
	if lookup == nil {
		lookup = triggerLookups["default"]
	}

	var modParts []string
	for _, r := range k.Modifier {
		modParts = append(modParts, lookup(string(r)))
	}
	mod := strings.Join(modParts, "-")

	var out string
	if mod != "" {
		out = mod + "-" + lookup(k.Key)
	} else {
		out = lookup(k.Key)
	}

	if k.Mode != "" {
		return k.Mode + "=" + out
	}
	return out
}

func formatBinding(b BindingEntry, program string) string {
	lookup := bindingLookups[normalizeProgram(program)]
	if lookup == nil {
		lookup = bindingLookups["default"]
	}

	key := b.Sequence
	if key == "" {
		key = b.Binding.Key
	}

	fmt.Println("DEBUG: ", b)

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

type Formatter struct {
	Trigger func(KeySeq, string) string
	Binding func(BindingEntry, string) string
}

type BindingFormatConfig map[string]map[string]string

func loadBindingFormat(path string) BindingFormatConfig {
	var cfg BindingFormatConfig
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		log.Fatalf("failed to load binding format config: %v", err)
	}
	return cfg
}

type TriggerFormatConfig map[string]map[string]string

func loadTriggerFormat(path string) TriggerFormatConfig {
	var cfg TriggerFormatConfig
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		log.Fatalf("failed to load trigger format config: %v", err)
	}
	return cfg
}

type TriggerLookup func(string) string

func buildLookupFuncs(cfg map[string]map[string]string) map[string]TriggerLookup {
	defaultMap := cfg["default"]
	out := make(map[string]TriggerLookup)

	for program, mapping := range cfg {
		local := mapping // capture per iteration

		out[program] = func(local map[string]string) TriggerLookup {
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
