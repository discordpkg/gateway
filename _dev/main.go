package main

import (
	"github.com/andersfylling/discordgateway/_dev/generate"
	"github.com/andersfylling/discordgateway/_dev/generate/commands"
	"github.com/andersfylling/discordgateway/_dev/generate/events"
	"github.com/andersfylling/discordgateway/_dev/generate/intents"
	"io/ioutil"
	"strings"
)

func DiscordConstantToGoNamingConvention(name string) string {
	words := strings.Split(name, "_")
	for i, word := range words {
		word = strings.ToLower(word)
		words[i] = strings.ToUpper(string(word[0])) + word[1:]
	}

	return strings.Join(words, "")
}

func main() {
	schematic := ReadDiscordSchematic("../discord.yaml")
	GenerateIntents(&schematic.Gateway)
	GenerateEvents(&schematic.Gateway)
	GenerateCommands(&schematic.Gateway)
}

func GenerateIntents(gateway *GatewayYAML) {
	rows := make([]*intents.IntentInfo, len(gateway.Intents))
	for i := range gateway.Intents {
		intent := gateway.Intents[i]

		intentEvents := make([]*intents.EventInfo, len(intent.Events))
		for j := range intent.Events {
			evt := intent.Events[j].ID
			intentEvents[j] = &intents.EventInfo{
				Name:  DiscordConstantToGoNamingConvention(evt),
				Event: evt,
			}
		}

		name := DiscordConstantToGoNamingConvention(intent.ID)
		rows[i] = &intents.IntentInfo{
			Name:      name,
			Intent:    intent.ID,
			BitOffset: intent.BitOffset,
			Events:    intentEvents,
			DM:        intent.DirectMessage,
		}
	}

	templateFilePath := "generate/intents/intents.gohtml"
	code, err := generate.GoCode(rows, templateFilePath)
	if err != nil {
		panic(err)
	}

	generatedFilePath := "../intent/intents_gen.go"
	if err = ioutil.WriteFile(generatedFilePath, code, 0644); err != nil {
		panic(err)
	}
}

func GenerateEvents(gateway *GatewayYAML) {
	rows := make([]*events.EventInfo, len(gateway.Events))
	for i := range gateway.Events {
		event := gateway.Events[i]

		name := DiscordConstantToGoNamingConvention(event.ID)
		rows[i] = &events.EventInfo{
			Name:        name,
			Event:       event.ID,
			Description: event.Description,
		}
	}

	templateFilePath := "generate/events/events.gohtml"
	code, err := generate.GoCode(rows, templateFilePath)
	if err != nil {
		panic(err)
	}

	generatedFilePath := "../event/events_gen.go"
	if err = ioutil.WriteFile(generatedFilePath, code, 0644); err != nil {
		panic(err)
	}
}

func GenerateCommands(gateway *GatewayYAML) {
	rows := make([]*commands.Info, len(gateway.Commands))
	for i := range gateway.Commands {
		command := gateway.Commands[i]

		name := DiscordConstantToGoNamingConvention(command.ID)
		rows[i] = &commands.Info{
			Name:        name,
			Opcode:      command.Opcode,
			Description: command.Description,
		}
	}

	templateFilePath := "generate/commands/commands.gohtml"
	code, err := generate.GoCode(rows, templateFilePath)
	if err != nil {
		panic(err)
	}

	generatedFilePath := "../command/commands_gen.go"
	if err = ioutil.WriteFile(generatedFilePath, code, 0644); err != nil {
		panic(err)
	}
}
