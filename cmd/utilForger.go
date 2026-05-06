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
	"strings"

	"github.com/DanielRivasMD/domovoi"
	"github.com/DanielRivasMD/horus"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

type moldReplace struct {
	old string
	new string
}

type moldForge struct {
	out      string
	files    []string
	replaces []moldReplace
}

func newMoldConfig(outFile string, inFiles []string, replaces ...moldReplace) moldForge {
	return moldForge{
		out:      outFile,
		files:    inFiles,
		replaces: replaces,
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func moldForging(op string, mf moldForge) {
	fmt.Println(mf.Cmd())
	horus.CheckErr(
		domovoi.ExecSh(mf.Cmd()),
		horus.WithOp(op),
		horus.WithCategory("shell_command"),
		horus.WithMessage("Failed to execute mbombo command"),
		horus.WithDetails(map[string]any{
			"command": mf.Cmd(),
		}),
	)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func replace(key, val string) moldReplace {
	return moldReplace{old: key, new: val}
}

func (m moldForge) Cmd() string {
	var files []string
	for _, f := range m.files {
		files = append(files, fmt.Sprintf(`--files %s`, f))
	}
	fileBlock := strings.Join(files, " \\\n")
	var replaces []string
	for _, r := range m.replaces {
		replaces = append(replaces, fmt.Sprintf(`--replace %s="%s"`, r.old, r.new))
	}
	replaceBlock := strings.Join(replaces, " \\\n")
	return fmt.Sprintf(
		`mbombo \
--out %s \
%s \
%s`,
		m.out,
		fileBlock,
		replaceBlock,
	)
}

////////////////////////////////////////////////////////////////////////////////////////////////////
