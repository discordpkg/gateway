package main

import (
	"fmt"
)

func main() {
	y := ReadDiscordSchematic(".../discord.yaml")
	fmt.Println(*y)
}
