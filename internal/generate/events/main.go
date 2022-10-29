package main

import (
	"sort"
	"strings"

	"github.com/discordpkg/gateway/internal/generate"
)

type EventData struct {
	Name  string
	Value string
}

func main() {
	events := parseEvents("internal/discord-api-docs/docs/topics/Gateway_Events.md")
	sort.Slice(events, func(i, j int) bool {
		return events[i].Name < events[j].Name
	})

	generate.Generate(events, "internal/generate/events/events_gen.go.tmpl", "event/events_gen.go")
}

func parseEvents(filePath string) []*EventData {
	tables := generate.ExtractMarkdownTables(filePath, func(name string) bool {
		return strings.ToLower(name) == "receive events" || strings.ToLower(name) == "send events"
	})
	if len(tables) != 2 {
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

	if len(events) < 20 {
		// 20 is kinda random, could be 30, we just want to be sure we have a bunch of events
		panic("why are there so few event types?")
	}
	return events
}
