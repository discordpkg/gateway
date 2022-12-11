package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/discordpkg/gateway/internal/generate"
)

type EventCodeData struct {
	Name string
	Code string
}

type EventData struct {
	Name  string
	Value string
}

func main() {
	receiveEvents := parseEvents("internal/discord-api-docs/docs/topics/Gateway_Events.md", "receive events")
	generate.Generate(receiveEvents, "internal/generate/events/events_gen.go.tmpl", "event/receive_gen.go")

	sendEvents := parseEvents("internal/discord-api-docs/docs/topics/Gateway_Events.md", "send events")
	generate.Generate(sendEvents, "internal/generate/events/events_gen.go.tmpl", "event/send_gen.go")

	// generate utils
	events := append(receiveEvents, sendEvents...)
	generate.Generate(events, "internal/generate/events/util_gen.go.tmpl", "event/utils_gen.go")

	eventCodes := parseEventCodes("internal/discord-api-docs/docs/topics/Opcodes_and_Status_Codes.md", "gateway opcodes")
	generate.Generate(eventCodes, "internal/generate/events/opcodes_gen.go.tmpl", "event/opcode/codes_gen.go")

	// generate method for converting event to opcode
	// remove opcodes which can't be mapped by name to an event
	for i := len(eventCodes) - 1; i >= 0; i-- {
		exists := false
		for j := range events {
			if eventCodes[i].Name == events[j].Name {
				exists = true
				break
			}
		}

		if !exists {
			fmt.Println("WARN: removed opcode from event method: " + eventCodes[i].Name)
			eventCodes[i] = eventCodes[len(eventCodes)-1]
			eventCodes = eventCodes[:len(eventCodes)-1]
		}
	}
	generate.Generate(eventCodes, "internal/generate/events/event_codes_gen.go.tmpl", "event/codes_gen.go")
}

func parseEventCodes(filePath string, tableName string) []*EventCodeData {
	tables := generate.ExtractMarkdownTables(filePath, generate.AnyTable([]string{tableName}))
	if len(tables) != 1 {
		panic("wrong amount of matches for 'gateway events opcodes' tables")
	}

	var events []*EventCodeData
	for _, table := range tables {
		for i := range table.Rows {
			row := table.Rows[i]

			name := strings.Join(strings.Split(row[1], " "), "")
			code := row[0]

			events = append(events, &EventCodeData{Name: name, Code: code})
		}
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].Name < events[j].Name
	})
	return events
}

func parseEvents(filePath string, tableName string) []*EventData {
	tables := generate.ExtractMarkdownTables(filePath, generate.AnyTable([]string{tableName}))
	if len(tables) != 1 {
		panic("wrong amount of matches for 'gateway events' tables")
	}

	// Iterate over all rows and get the actual event value.
	// Assumptions:
	//  1. Every row uses a markdown link such as: [Hello](#DOCS_TOPICS_GATEWAY/hello)
	//  2. Values are title case with space between each word
	var events []*EventData
	for _, table := range tables {
		for i := range table.Rows {
			row := table.Rows[i]
			markdownLink := row[0]
			eventTitle := strings.Split(markdownLink[1:], "]")[0]
			name := strings.Join(strings.Split(eventTitle, " "), "")
			value := strings.Join(strings.Split(strings.ToUpper(eventTitle), " "), "_")

			events = append(events, &EventData{Name: name, Value: value})
		}
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].Name < events[j].Name
	})
	return events
}
