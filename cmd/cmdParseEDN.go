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

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/edn-format/edn"

	"github.com/spf13/cobra"
	"github.com/ttacon/chalk"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

// configurable variables
const (
	TcPrefix    = "TC"    // change this to modify the EDN prefix
	DefaultKey  = " "     // display for unmapped keys
	OutputDir   = "layouts"
	OutputFile  = "keyboard_layout.md"
	EdnFilePath = "keyboard_config.edn"
)

type KeyboardConfig struct {
	Letters      map[string]string
	SpecialKeys  map[string]string
	UsedTcPrefix string
}

var ()

////////////////////////////////////////////////////////////////////////////////////////////////////

// parseednCmd
var parseednCmd = &cobra.Command{
	Use:   "parseedn",
	Short: "" + chalk.Yellow.Color("") + ".",
	Long: chalk.Green.Color(chalk.Bold.TextStyle("Daniel Rivas ")) + chalk.Dim.TextStyle(chalk.Italic.TextStyle("<danielrivasmd@gmail.com>")) + `
`,

	Example: `
` + chalk.Cyan.Color("babel") + ` help ` + chalk.Yellow.Color("") + chalk.Yellow.Color("parseedn"),

	////////////////////////////////////////////////////////////////////////////////////////////////////

}

////////////////////////////////////////////////////////////////////////////////////////////////////

// execute prior main
func init() {
	rootCmd.AddCommand(parseednCmd)

	// flags
}

////////////////////////////////////////////////////////////////////////////////////////////////////


// func main() {
// 	config := parseEdnConfig(EdnFilePath)
// 	generateMarkdown(config)
// 	fmt.Printf("Generated layout using TC variable: '%s'\n", TcPrefix)
// 	fmt.Printf("Output: %s/%s\n", OutputDir, OutputFile)
// }

func parseEdnConfig(filePath string) KeyboardConfig {
	file, err := os.ReadFile(filePath)
	if err != nil {
		panic(fmt.Sprintf("Error reading EDN file: %v", err))
	}

	var data map[string]interface{}
	if err := edn.Unmarshal(file, &data); err != nil {
		panic(fmt.Sprintf("Error parsing EDN: %v", err))
	}

	config := KeyboardConfig{
		Letters:     make(map[string]string),
		SpecialKeys: make(map[string]string),
	}

	// initialize all letter keys
	for c := 'a'; c <= 'z'; c++ {
		config.Letters[string(c)] = DefaultKey
	}

	// initialize special keys
	specialKeys := []string{
		"open_bracket", "close_bracket", "semicolon", "quote",
		"backslash", "comma", "period", "slash",
		"delete_or_backspace", "return_or_enter",
		"right_shift", "right_option", "right_command", "spacebar",
	}
	for _, key := range specialKeys {
		config.SpecialKeys[key] = DefaultKey
	}
	config.SpecialKeys["delete_or_backspace"] = "BACK"
	config.SpecialKeys["return_or_enter"] = "ENTER"
	config.SpecialKeys["right_shift"] = "SHIFT"
	config.SpecialKeys["right_option"] = "ALT"
	config.SpecialKeys["right_command"] = "CMD"
	config.SpecialKeys["spacebar"] = "SPACE"

	// parse rules
	rules, ok := data[":rules"].([]interface{})
	if !ok {
		return config
	}

	for _, rule := range rules {
		ruleList, ok := rule.([]interface{})
		if !ok || len(ruleList) < 2 {
			continue
		}

		key, ok := ruleList[0].(edn.Keyword)
		if !ok {
			continue
		}

		keyStr := string(key)
		value := ruleList[1]

		// handle letter keys (a-z)
		for c := 'a'; c <= 'z'; c++ {
			letter := string(c)
			if keyStr == fmt.Sprintf(":!%s#P%s", TcPrefix, letter) {
				config.Letters[letter] = formatEdnValue(value)
				break
			}
		}

		// handle special keys
		switch keyStr {
		case fmt.Sprintf(":!%s#Popen_bracket", TcPrefix):
			config.SpecialKeys["open_bracket"] = formatEdnValue(value)
		case fmt.Sprintf(":!%s#Pclose_bracket", TcPrefix):
			config.SpecialKeys["close_bracket"] = formatEdnValue(value)
		// Add other special key cases...
		case fmt.Sprintf(":!%s#Pdelete_or_backspace", TcPrefix):
			config.SpecialKeys["delete_or_backspace"] = formatEdnValue(value)
		}
	}

	config.UsedTcPrefix = TcPrefix
	return config
}

func formatEdnValue(value interface{}) string {
	switch v := value.(type) {
	case []interface{}:
		var parts []string
		for _, item := range v {
			parts = append(parts, fmt.Sprint(item))
		}
		return strings.Join(parts, " ")
	case edn.Keyword:
		return string(v)
	default:
		return fmt.Sprint(v)
	}
}

func generateMarkdown(config KeyboardConfig) {
	// create output directory
	if err := os.MkdirAll(OutputDir, 0755); err != nil {
		panic(fmt.Sprintf("Error creating output directory: %v", err))
	}

	file, err := os.Create(fmt.Sprintf("%s/%s", OutputDir, OutputFile))
	if err != nil {
		panic(fmt.Sprintf("Error creating output file: %v", err))
	}
	defer file.Close()

	// helper function to center text
	center := func(text string, width int) string {
		if len(text) >= width {
			return text
		}
		padding := (width - len(text)) / 2
		return fmt.Sprintf("%*s%s%*s", padding, "", text, padding, "")
	}

	// generate the markdown content
	content := fmt.Sprintf(`# Dynamic Keyboard Layout
*Generated with TC='%s'*

`+"```markdown"+`
┌─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬───────────┐
| ~ ` | ! 1 | @ 2 | # 3 | $ 4 | %% 5 | ^ 6 | & 7 | * 8 | ( 9 | ) 0 | _ - | + = | %s |
| TAB | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |
| CAPS | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |      %s      |
| SHIFT  | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |     %s     |
| CTRL | ALT | CMD │               %s               │ %s | %s │
└─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴───────────┘
`+"```"+`

### Active Mappings
- **Letters**: %s
- **Specials**: %s (SPACE), %s (ENTER)
- **TC Variable**: '%s' (change in script)
`,
		config.UsedTcPrefix,
		center(config.SpecialKeys["delete_or_backspace"], 8),
		center(config.Letters["q"], 3), center(config.Letters["w"], 3), center(config.Letters["e"], 3),
		center(config.Letters["r"], 3), center(config.Letters["t"], 3), center(config.Letters["y"], 3),
		center(config.Letters["u"], 3), center(config.Letters["i"], 3), center(config.Letters["o"], 3),
		center(config.Letters["p"], 3), config.SpecialKeys["open_bracket"], config.SpecialKeys["close_bracket"],
		center(config.SpecialKeys["backslash"], 8),
		center(config.Letters["a"], 3), center(config.Letters["s"], 3), center(config.Letters["d"], 3),
		center(config.Letters["f"], 3), center(config.Letters["g"], 3), center(config.Letters["h"], 3),
		center(config.Letters["j"], 3), center(config.Letters["k"], 3), center(config.Letters["l"], 3),
		config.SpecialKeys["semicolon"], config.SpecialKeys["quote"], center(config.SpecialKeys["return_or_enter"], 8),
		center(config.Letters["z"], 3), center(config.Letters["x"], 3), center(config.Letters["c"], 3),
		center(config.Letters["v"], 3), center(config.Letters["b"], 3), center(config.Letters["n"], 3),
		center(config.Letters["m"], 3), config.SpecialKeys["comma"], config.SpecialKeys["period"],
		config.SpecialKeys["slash"], center(config.SpecialKeys["right_shift"], 8),
		center(config.SpecialKeys["spacebar"], 16), config.SpecialKeys["right_command"],
		config.SpecialKeys["right_option"],
		getActiveMappings(config.Letters),
		config.SpecialKeys["spacebar"], config.SpecialKeys["return_or_enter"],
		config.UsedTcPrefix,
	)

	file.WriteString(content)
}

func getActiveMappings(letters map[string]string) string {
	var active []string
	for c := 'a'; c <= 'z'; c++ {
		letter := string(c)
		if letters[letter] != DefaultKey {
			active = append(active, fmt.Sprintf("%s: %s", letter, letters[letter]))
		}
	}
	if len(active) == 0 {
		return "None"
	}
	return strings.Join(active, ", ")
}

////////////////////////////////////////////////////////////////////////////////////////////////////
