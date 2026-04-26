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
	"io"
	"log"
	"strings"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

///////////////////////////////////////////////////////////////////////////////////////////////////

func emitConfig(w io.Writer, entries []BindingEntry, target string) {
	filtered := filterByProgram(entries, target)
	rawBind := make(map[string]string)
	for _, entry := range filtered {
		for _, actions := range entry.Actions {
			bindKey := formatKeySeq(entry.Binding, lookups.interpret, actions.Program, "-")
			rawBind[bindKey] = actions.Command
		}
	}
	formatted := formatBinds(rawBind, target)
	switch {
	case strings.HasPrefix(target, "helix-"):
		if headerLines, ok := programHeaders[target]; ok {
			for _, line := range headerLines {
				fmt.Fprintln(w, line)
			}
		}
		for key, val := range formatted {
			fmt.Fprintf(w, "%s = %s\n", key, val)
		}
	case target == "micro":
		fmt.Fprintln(w, "{")
		if headerLines, ok := programHeaders[target]; ok {
			for _, line := range headerLines {
				fmt.Fprintln(w, line)
			}
		}
		for key, val := range formatted {
			fmt.Fprintf(w, "  %q: %q,\n", key, val)
		}
		fmt.Fprintln(w, "}")
	default:
		log.Fatalf("unsupported --program %q", target)
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////
