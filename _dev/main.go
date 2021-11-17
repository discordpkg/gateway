package main

import (
	"fmt"
)

func main() {
	y := ReadDiscordYaml(".../discord.yaml")
	fmt.Println(*y)
}
