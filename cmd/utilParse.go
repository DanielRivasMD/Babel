////////////////////////////////////////////////////////////////////////////////////////////////////

package cmd

////////////////////////////////////////////////////////////////////////////////////////////////////

import (
	"fmt"
	"os"
	"strings"

	"github.com/ttacon/chalk"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

// Package-level variable for the TC prefix.
// TODO: reuse parse functions to render keyboard
var TC = "TC"

// ednFile is assumed to be declared externally.

const (
	DefaultKey = " "
	OutputDir  = "layouts"
	OutputFile = "keyboard_layout.md"
)

// mappingLabels converts an EDN rule key (without any prefix markers) to a friendly label.
var mappingLabels = map[string]string{
	"hyphen":              "-",
	"equal_sign":          "=",
	"delete_or_backspace": "BACK",
	"return_or_enter":     "ENTER",
	"right_shift":         "SHIFT",
	"right_option":        "ALT",
	"right_command":       "CMD",
	"spacebar":            "SPACE",
	// TODO: add lefts
}

type KeyConfig struct {
	key       string
	kode      string
	interpret string
	app       string
	commented bool
	term      []Term
}

type Term struct {
	app         string
	description string
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func parse() {
	// TODO: high level: parse edn => read file line by line to extract values. mount on structs to indicate config & color
	// TODO: high level: generate markdown => extract values mounted on structs

	// Initialize the configuration.
	config := initConfig()

	// For example, assume filePath is passed in or defined here.
	// filePath := "your_edn_file.edn"
	if err := updateConfigFromFile(config, flags.ednFile); err != nil {
		fmt.Printf("Error reading EDN file: %v\n", err)
		return
	}

	generateMarkdown(config)

	if flags.verbose {
		fmt.Printf("Generated layout using TC variable: '%s'\n", TC)
		fmt.Printf("Output: %s/%s\n", OutputDir, OutputFile)
	}

}

////////////////////////////////////////////////////////////////////////////////////////////////////

func NewKeyConfig(key string) KeyConfig {
	return KeyConfig{
		key:       key,
		kode:      "",
		interpret: "",
		app:       "",
		commented: false,
		term:      []Term{},
	}
}

func initConfig() map[string]KeyConfig {
	// Initialize configuration.
	config := make(map[string]KeyConfig)

	// Initialize letter keys (a-z) with default.
	for c := 'a'; c <= 'z'; c++ {
		config[string(c)] = NewKeyConfig(string(c))
	}

	// Initialize number keys: digits 1-0, dash and equals.
	digitKeys := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "0"}
	for _, d := range digitKeys {
		config[d] = NewKeyConfig(d)
	}

	// Initialize special keys.
	specialKeys := []string{
		"hyphen", "equal_sign",
		"open_bracket", "close_bracket",
		"semicolon", "quote", "backslash",
		"comma", "period", "slash",
		"delete_or_backspace", "return_or_enter",
		"right_shift", "right_option", "right_command", "spacebar",
		"left_arrow", "right_arrow", "up_arrow", "down_arrow",
	}
	for _, k := range specialKeys {
		config[k] = NewKeyConfig(k)
	}

	return config
}

// updateConfigFromFile reads the EDN file and updates the configuration map.
// It looks for lines that include "  [:!TC#P" and splits on spaces and semicolons.
// The key is extracted by splitting the third field on "#P", the fourth whitespace field
// is assigned to the KeyConfig.kode, and if the line (split by semicolons) has 3 or more fields,
// the line is considered "commented".
func updateConfigFromFile(config map[string]KeyConfig, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.Contains(line, "  [:!"+TC+"#P") {

			// Split by whitespace.
			fieldsSpace := strings.Fields(line)
			if len(fieldsSpace) < 4 {
				continue // not enough fields; skip.
			}

			// keys[2] should be like ":!TC#P<key>".
			parts := strings.Split(fieldsSpace[0], "#P")
			if len(parts) < 2 {
				continue
			}

			key := parts[1]
			key = strings.TrimSuffix(key, "]")
			// Check for comment: if splitting the original line by ';' yields 3 or more fields.
			fieldsSemi := strings.Split(line, ";")

			hasComment := len(fieldsSemi) >= 3

			// We must fetch the KeyConfig, modify it, then reassign it.
			if kc, ok := config[key]; ok {

				kode := fieldsSpace[1]
				kode = strings.TrimSuffix(kode, "]")
				kode = strings.TrimSuffix(kode, "]")
				kode = strings.TrimPrefix(kode, "[:")
				kode = strings.TrimPrefix(kode, "!")

				kc.commented = hasComment
				if kc.commented {
					kc.kode = chalk.Bold.TextStyle(chalk.Yellow.Color(kode))
				} else {
					kc.kode = chalk.Bold.TextStyle(chalk.Cyan.Color(kode))
				}
				config[key] = kc
			} else {
				// Optionally, handle keys not present in the map.
				// For now, we simply ignore them.
			}
		}
	}
	return nil
}

func buildCommentMap(filePath string) map[string]bool {
	commentMap := make(map[string]bool)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return commentMap // return an empty map on error
	}
	lines := strings.Split(string(data), "\n")
	// This regex captures the key following "[:!TC#P" until a space, semicolon, or closing bracket.
	for _, line := range lines {
		if strings.Contains(line, "  [:!"+TC+"#P") {
			keys := strings.Split(line, " ")
			if len(keys) >= 2 {
				key := strings.Split(keys[2], "#P")[1]
				fields := strings.Split(line, ";")
				// If there are more than three fields, we consider the rule as having an extra comment.
				hasComment := len(fields) >= 3
				commentMap[key] = hasComment
			}
		}
	}
	return commentMap
}

// derivedKey converts a target like "Phyphen" or "Popen_bracket" into the final config key.
func derivedKey(target string) string {
	if strings.HasPrefix(target, "P") {
		return strings.TrimPrefix(target, "P")
	}
	return target
}

// extractMappingComments extracts mapping comments using a regular expression that captures the key name.
func extractMappingComments(filePath string) []string {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}
	rawText := string(data)
	lines := strings.Split(rawText, "\n")
	var comments []string
	for _, line := range lines {
		if strings.Contains(line, "[:!TC#P") {
			parts := strings.Split(line, ";")
			if len(parts) >= 3 {
				commentVal := strings.TrimSpace(parts[1])
				commentKey := strings.TrimSpace(parts[2])
				if commentVal != "" && commentKey != "" {
					comments = append(comments, fmt.Sprintf("%s => %s", commentKey, commentVal))
				}
			}
		}
	}
	return comments
}

func generateMarkdown(config map[string]KeyConfig) {
	if err := os.MkdirAll(OutputDir, 0755); err != nil {
		panic(fmt.Sprintf("Error creating output directory: %v", err))
	}

	file, err := os.Create(fmt.Sprintf("%s/%s", OutputDir, OutputFile))
	if err != nil {
		panic(fmt.Sprintf("Error creating output file: %v", err))
	}
	defer file.Close()

	center := func(text string, width int) string {
		if len(text) >= width {
			return text
		}
		padding := (width - len(text)) / 2
		return fmt.Sprintf("%*s%s%*s", padding, "", text, padding, "")
	}

	markdownStart := fmt.Sprintf(`# Dynamic Keyboard Layout
*Generated*

`)
	codeFenceStart := "```markdown\n"
	codeFenceEnd := "```\n"

	// Build the dynamic number row. The final cell uses the value for delete_or_backspace.
	numberRow := fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |",
		"~ `",
		config["1"].kode, config["2"].kode, config["3"].kode,
		config["4"].kode, config["5"].kode, config["6"].kode,
		config["7"].kode, config["8"].kode, config["9"].kode,
		config["0"].kode, config["hyphen"].kode, config["equal_sign"].kode,
		config["delete_or_backspace"].kode,
	)

	topBorder := "┌─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬───────────┐\n"
	secondRow := fmt.Sprintf("| TAB | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
		center(config["q"].kode, 3), center(config["w"].kode, 3), center(config["e"].kode, 3),
		center(config["r"].kode, 3), center(config["t"].kode, 3), center(config["y"].kode, 3),
		center(config["u"].kode, 3), center(config["i"].kode, 3), center(config["o"].kode, 3),
		center(config["p"].kode, 3), config["open_bracket"].kode, config["close_bracket"].kode,
		center(config["backslash"].kode, 8),
	)

	thirdRow := fmt.Sprintf("| CAPS | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |      %s      |\n",
		center(config["a"].kode, 3), center(config["s"].kode, 3), center(config["d"].kode, 3),
		center(config["f"].kode, 3), center(config["g"].kode, 3), center(config["h"].kode, 3),
		center(config["j"].kode, 3), center(config["k"].kode, 3), center(config["l"].kode, 3),
		config["semicolon"].kode, config["quote"].kode, center(config["return_or_enter"].kode, 8),
	)
	fourthRow := fmt.Sprintf("| SHIFT  | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |     %s     |\n",
		center(config["z"].kode, 3), center(config["x"].kode, 3), center(config["c"].kode, 3),
		center(config["v"].kode, 3), center(config["b"].kode, 3), center(config["n"].kode, 3),
		center(config["m"].kode, 3), config["comma"].kode, config["period"].kode,
		config["slash"].kode, center(config["right_shift"].kode, 8),
	)
	fifthRow := fmt.Sprintf("| CTL | ALT | CMD │               %s               │ %s | %s │\n",
		center(config["spacebar"].kode, 16),
		config["right_command"].kode,
		config["right_option"].kode,
	)
	bottomBorder := "└─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴───────────┘\n"

	finalLayout := topBorder + numberRow + "\n" + secondRow + thirdRow + fourthRow + fifthRow + bottomBorder

	// activeMappingsSection := fmt.Sprintf("\n### Active Mappings\n- **Letters**: %s\n- **Specials**: %s (SPACE), %s (ENTER)\n- **Arrows**: %s\n- **TC Variable**: '%s' (change in script)\n",
	// 	getActiveMappings(config.Letters),
	// 	config.SpecialKeys["spacebar"],
	// 	config.SpecialKeys["return_or_enter"],
	// 	getArrowMappings(config),
	// 	config.UsedTcPrefix,
	// )

	mappingComments := extractMappingComments(flags.ednFile)
	mappingCommentsSection := ""
	if len(mappingComments) > 0 {
		mappingCommentsSection = "\n### Mapping Comments\n"
		for _, comment := range mappingComments {
			mappingCommentsSection += "- " + comment + "\n"
		}
	}

	finalContent := markdownStart + codeFenceStart + finalLayout + codeFenceEnd + mappingCommentsSection

	if _, writeErr := file.WriteString(finalContent); writeErr != nil {
		panic(fmt.Sprintf("Error writing to output file: %v", writeErr))
	}
}

// func getActiveMappings(letters map[string]string) string {
// 	var active []string
// 	for c := 'a'; c <= 'z'; c++ {
// 		letter := string(c)
// 		if letters[letter] != DefaultKey {
// 			active = append(active, fmt.Sprintf("%s: %s", letter, letters[letter]))
// 		}
// 	}
// 	if len(active) == 0 {
// 		return "None"
// 	}
// 	return strings.Join(active, ", ")
// }

// func getArrowMappings(config KeyboardConfig) string {
// 	arrows := []string{"left_arrow", "right_arrow", "up_arrow", "down_arrow"}
// 	var mappings []string
// 	for _, arrow := range arrows {
// 		if val, ok := config.SpecialKeys[arrow]; ok && val != DefaultKey {
// 			mappings = append(mappings, fmt.Sprintf("%s: %s", arrow, val))
// 		}
// 	}
// 	if len(mappings) == 0 {
// 		return "None"
// 	}
// 	return strings.Join(mappings, ", ")
// }

////////////////////////////////////////////////////////////////////////////////////////////////////
