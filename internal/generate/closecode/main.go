package main

import (
	"github.com/discordpkg/gateway/internal/generate"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type CloseCodeData struct {
	Name      string
	Code      int
	Reconnect bool
}

var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9 ]+`)

func clearString(str string) string {
	return nonAlphanumericRegex.ReplaceAllString(str, "")
}

func main() {
	events := parseCloseCodeData("internal/discord-api-docs/docs/topics/Opcodes_and_Status_Codes.md")
	sort.Slice(events, func(i, j int) bool {
		return events[i].Name < events[j].Name
	})

	generate.Generate(events, "internal/generate/closecode/closecodes_gen.go.tmpl", "closecode/closecodes_gen.go")
}

func parseCloseCodeData(filePath string) []*CloseCodeData {
	tables := generate.ExtractMarkdownTables(filePath, func(name string) bool {
		return strings.ToLower(name) == "gateway close event codes"
	})
	if len(tables) != 1 {
		panic("wrong amount of matches for 'gateway close event codes' tables")
	}
	table := tables[0]

	// Iterate over all rows and get the actual opcodes.
	var closeCodes []*CloseCodeData
	for i := range table.Rows {
		row := table.Rows[i]
		code, err := strconv.Atoi(strings.Trim(row[0], " "))
		if err != nil {
			panic("unable to parse close code to int: " + err.Error())
		}

		name := strings.Title(clearString(strings.Trim(row[1], " ")))
		name = strings.Join(strings.Split(name, " "), "")
		reconnect := strings.ToLower(strings.Trim(row[3], " ")) == "true"

		closeCodes = append(closeCodes, &CloseCodeData{Name: name, Code: code, Reconnect: reconnect})
	}

	if len(closeCodes) < 5 {
		panic("why are there so few close codes?")
	}
	return closeCodes
}
