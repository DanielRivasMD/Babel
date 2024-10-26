/*
Copyright © 2024 Daniel Rivas <danielrivasmd@gmail.com>

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
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// docCmd represents the doc command
var docCmd = &cobra.Command{
	Use:   "doc",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("doc called")
	},
}

func init() {
	rootCmd.AddCommand(docCmd)

	target := findHome() + "/" + ".saiyajin/karabiner/karabiner.edn"
	// params := copyCR(target, "test.txt")

	// open reader
	fread, ε := os.Open(target)
	if ε != nil {
		log.Fatal(ε)
	}
	defer fread.Close()

	// // open writer
	// fwrite, ε := os.OpenFile(target, os.O_WRONLY|os.O_CREATE, 0666)
	// if ε != nil {
	// 	log.Fatal(ε)
	// }
	// defer fwrite.Close()

	// read file
	scanner := bufio.NewScanner(fread)

	// scan file
	for scanner.Scan() {

	if strings.HasPrefix(scanner.Text(), "  [") {
		fmt.Println(scanner.Text())

		// tab separated records
		records := strings.Split(scanner.Text(), " ")

		// fmt.Println(records)
		// fmt.Println(records[0])
		// fmt.Println(records[1])

		fr := records[2]
		fr = strings.Replace(fr, "[:!", "", -1)
		fr = strings.Replace(fr, "#P", "", -1)
		fr = strings.Replace(fr, "O", "alt-", -1)
		fr = strings.Replace(fr, "T", "ctl-", -1)
		fr = strings.Replace(fr, "C", "cmd-", -1)
		fmt.Println(fr)

		to := records[3]
		to = strings.Replace(to, ":!", "", -1)
		to = strings.Replace(to, "S", "shift", -1)
		fmt.Println(to)

		fmt.Println(records[4])

		// identify potential lines
		
	}

		// // write
		// _, ε = ϖ.WriteString(toPrint)
		// if ε != nil {
		// 	log.Fatal(ε)
		// }
	}

	if ε := scanner.Err(); ε != nil {
		log.Fatal(ε)
	}

	// // flush writer
	// ϖ.Flush()

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// docCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// docCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
