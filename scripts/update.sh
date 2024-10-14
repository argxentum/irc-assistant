#!/usr/bin/env bash

echo "Pulling latest changes"
git pull

if [ "$1" == "--irc" ]; then
	echo "Rebuilding assistant"
	go build -o bin/assistant assistant/cmd/assistant
elif [ "$1" == "--web" ]; then
	echo "Rebuilding assistant-web"
	go build -o bin/assistant-web assistant/cmd/assistant-web
elif [ "$1" == "--scheduler" ]; then
	echo "Rebuilding assistant-scheduler"
	go build -o bin/assistant-scheduler assistant/cmd/assistant-scheduler
else
	echo "Rebuilding assistant, assistant-web, and assistant-scheduler"
	go build -o bin/assistant assistant/cmd/assistant
	go build -o bin/assistant-web assistant/cmd/assistant-web
	go build -o bin/assistant-scheduler assistant/cmd/assistant-scheduler
fi