/*
Copyright © 2026 Daniel Rivas <danielrivasmd@gmail.com>

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
	"os"
	"strings"

	"github.com/DanielRivasMD/horus"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

func compose(op string) {
	if rootFlags.program != "kanata" {
		horus.CheckErr(
			fmt.Errorf("unsupported program %q for compose", rootFlags.program),
			horus.WithOp(op),
			horus.WithExitCode(1),
		)
	}

	prefixes := []string{
		"T", "TS", "O", "OS", "C", "CS", "J", "S", "R", "Q", "QR", "E", "ER", "W", "WS", "tab", "q", "w", "z", "zS",
	}

	var suffixes []string
	for i := 1; i <= 9; i++ {
		suffixes = append(suffixes, fmt.Sprintf("%d", i))
	}
	suffixes = append(suffixes, "0")
	for c := 'a'; c <= 'z'; c++ {
		suffixes = append(suffixes, string(c))
	}
	suffixes = append(suffixes,
		"lf", "rg", "up", "dn",
		"hy", "eq", "db",
		"ob", "cb",
		"sc", "qu", "bl",
		"cm", "pe", "sl",
		"ret", "spc",
		"kR", "kE", "kQ", "kC", "kO", "kT", "kS", "kW",
	)

	var out strings.Builder
	out.WriteString("(defalias\n")

	for i, p := range prefixes {
		for _, s := range suffixes {
			key := p + s
			line := fmt.Sprintf("  %s", key)
			padding := 10 - len(line)
			if padding < 1 {
				padding = 1
			}
			line += strings.Repeat(" ", padding) + "XX\n"
			out.WriteString(line)
		}
		if i+1 != len(prefixes) {
			out.WriteString("\n")
		}
	}
	out.WriteString(")\n")

	if composeFlags.template != "" {
		err := os.WriteFile(composeFlags.template, []byte(out.String()), 0644)
		horus.CheckErr(err, horus.WithOp(op), horus.WithMessage("writing template file"))
	} else {
		fmt.Print(out.String())
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////
