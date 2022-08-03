package main

import (
	"github.com/discordpkg/gateway/internal/generate"
	"sort"
	"strconv"
	"strings"
)

type OpCodeData struct {
	Name    string
	Code    int
	Send    bool
	Receive bool
}

func main() {
	events := parseOpCodes("/home/anders/dev/gateway/internal/discord-api-docs/docs/topics/Opcodes_and_Status_Codes.md")
	sort.Slice(events, func(i, j int) bool {
		return events[i].Name < events[j].Name
	})

	generate.Generate(events, "internal/generate/opcode/opcodes_gen.go.tmpl", "opcode/opcodes_gen.go")
}

func parseOpCodes(filePath string) []*OpCodeData {
	tables := generate.ExtractMarkdownTables(filePath, func(name string) bool {
		return strings.ToLower(name) == "gateway opcodes"
	})
	if len(tables) != 1 {
		panic("wrong amount of matches for 'gateway opcodes' tables")
	}
	table := tables[0]

	// Iterate over all rows and get the actual opcodes.
	var opcodes []*OpCodeData
	for i := range table.Rows {
		row := table.Rows[i]
		code, err := strconv.Atoi(strings.Trim(row[0], " "))
		if err != nil {
			panic("unable to parse opcode to int: " + err.Error())
		}

		name := strings.Join(strings.Split(strings.Trim(row[1], " "), " "), "")
		directions := strings.Split(strings.ToLower(strings.Trim(row[2], " ")), "/")
		directionsMap := make(map[string]int)
		for _, direction := range directions {
			directionsMap[direction] = 0
		}

		_, send := directionsMap["send"]
		_, receive := directionsMap["receive"]
		opcodes = append(opcodes, &OpCodeData{Name: name, Code: code, Send: send, Receive: receive})
	}

	if len(opcodes) < 5 {
		panic("why are there so few opcodes?")
	}
	return opcodes
}
