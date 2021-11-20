package main

import (
	"github.com/andersfylling/discordgateway/_dev/generate"
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
