#!/bin/sh

docker build -t andersfylling/discordgateway-bot:latest -f cmd/discordgateway-alert-bot/Dockerfile .
docker push andersfylling/discordgateway-bot:latest

echo " "
echo "https://hub.docker.com/andersfylling/discordgateway-bot"