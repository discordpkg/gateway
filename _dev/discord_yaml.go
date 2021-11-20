package main

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v2"
	_ "gopkg.in/yaml.v2"
	"os"
	"sort"
)

type DiscordYAML struct {
	Gateway GatewayYAML `yaml:"gateway"`
}

func (d *DiscordYAML) Validate() error {
	validate := validator.New()
	if err := validate.Struct(d); err != nil {
		return err
	}

	for _, intent := range d.Gateway.Intents {
		for _, event := range intent.Events {
			if !d.Gateway.HasEvent(event.ID) {
				return fmt.Errorf(`intent "%s" refers to unknown event id: "%s"`, intent.ID, event.ID)
			}
		}
	}

	if err := d.Gateway.Validate(); err != nil {
		return err
	}
	return nil
}

func (d *DiscordYAML) Sort() {
	d.Gateway.Sort()
}

type GatewayYAML struct {
	Events   []GatewayEventYAML  `yaml:"events"`
	Commands []GatewayEventYAML  `yaml:"commands"`
	Intents  []GatewayIntentYAML `yaml:"intents"`
	Url      string              `yaml:"url" validate:"url"`
}

func (g *GatewayYAML) Validate() error {
	seen := make(map[string]bool)
	for _, event := range g.Events {
		if _, ok := seen[event.ID]; ok {
			return fmt.Errorf(`event "%s" must be unique`, event.ID)
		}
		seen[event.ID] = true
	}

	seen = make(map[string]bool)
	for _, intent := range g.Intents {
		if _, ok := seen[intent.ID]; ok {
			return fmt.Errorf(`intent "%s" must be unique`, intent.ID)
		}
		seen[intent.ID] = true
	}

	seenBitOffset := make(map[int]bool)
	for _, intent := range g.Intents {
		if _, ok := seenBitOffset[intent.BitOffset]; ok {
			return fmt.Errorf(`intent "%s" has a bit offset that is not unique`, intent.ID)
		}
		seenBitOffset[intent.BitOffset] = true
	}

	for _, intent := range g.Intents {
		if err := intent.Validate(); err != nil {
			return fmt.Errorf(`intent "%s" contains duplicate events: %w`, intent.ID, err)
		}
	}
	return nil
}

func (g *GatewayYAML) Sort() {
	sortEvents := func(events []GatewayEventYAML) {
		sort.Slice(events, func(i, j int) bool {
			return events[i].ID < events[j].ID
		})
	}

	sortEvents(g.Events)
	sort.Slice(g.Commands, func(i, j int) bool {
		return g.Commands[i].ID < g.Commands[j].ID
	})
	sort.Slice(g.Intents, func(i, j int) bool {
		return g.Intents[i].ID < g.Intents[j].ID
	})

	for _, intent := range g.Intents {
		sortEvents(intent.Events)
	}
}

func (g *GatewayYAML) HasEvent(id string) bool {
	for _, event := range g.Events {
		if event.ID == id {
			return true
		}
	}
	return false
}

type GatewayEventYAML struct {
	ID          string `yaml:"id" validate:"required,ascii,uppercase"`
	Description string `yaml:"description" validate:"ascii"`
	Url         string `yaml:"url" validate:"url"`

	// ShardID is specified for event that only work for one specific shard id
	ShardID *int `yaml:"shard_id" validate:"gte=0"`
}

type GatewayIntentYAML struct {
	ID          string             `yaml:"id" validate:"required,ascii,uppercase"`
	Description string             `yaml:"description" validate:"ascii"`
	Url         string             `yaml:"url" validate:"url"`
	BitOffset   int                `yaml:"bit_offset" validate:"gte=0"`
	Events      []GatewayEventYAML `yaml:"events" validate:"required"`

	// ShardID is specified for intents that only work for one specific shard id
	ShardID *int `yaml:"shard_id" validate:"gte=0"`

	DirectMessage bool `yaml:"direct_message"`
}

func (g *GatewayIntentYAML) Validate() error {
	seen := make(map[string]bool)
	for _, event := range g.Events {
		if _, ok := seen[event.ID]; ok {
			return fmt.Errorf(`event "%s" must be unique`, event.ID)
		}
		seen[event.ID] = true
	}
	return nil
}

func ReadDiscordSchematic(path string) *DiscordYAML {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	y := &DiscordYAML{}
	if err := yaml.Unmarshal(data, y); err != nil {
		panic(err)
	}

	if err := y.Validate(); err != nil {
		panic(err)
	}

	y.Sort()
	return y
}
