package main

import (
	"github.com/discordpkg/gateway/internal/generate"
	"sort"
	"strings"
)

type EventData struct {
	Name  string
	Value string
}

func main() {
	events := parseEvents("internal/discord-api-docs/docs/topics/Gateway.md")
	sort.Slice(events, func(i, j int) bool {
		return events[i].Name < events[j].Name
	})

	generate.Generate(events, "internal/generate/events/events_gen.go.tmpl", "event/events_gen.go")
}

func parseEvents(filePath string) []*EventData {
	tables := generate.ExtractMarkdownTables(filePath, func(name string) bool {
		return strings.ToLower(name) == "gateway events"
	})
	if len(tables) != 1 {
		panic("wrong amount of matches for 'gateway events' tables")
	}
	table := tables[0]

	// Iterate over all rows and get the actual event value.
	// Assumptions:
	//  1. Every row uses a markdown link such as: [Hello](#DOCS_TOPICS_GATEWAY/hello)
	//  2. Values are title case with space between each word
	var events []*EventData
	for i := range table.Rows {
		row := table.Rows[i]
		markdownLink := row[0]
		eventTitle := strings.Split(markdownLink[1:], "]")[0]
		name := strings.Join(strings.Split(eventTitle, " "), "")
		value := strings.Join(strings.Split(strings.ToUpper(eventTitle), " "), "_")

		events = append(events, &EventData{Name: name, Value: value})
	}

	if len(events) < 20 {
		// 20 is kinda random, could be 30, we just want to be sure we have a bunch of events
		panic("why are there so few event types?")
	}
	return events
}
