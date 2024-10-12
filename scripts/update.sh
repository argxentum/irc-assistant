#!/usr/bin/env bash

git pull

if [ "$1" == "--irc" ]; then
	echo "Rebuilding assistant"
	go build -o bin/assistant assistant/cmd/assistant
elif [ "$1" == "--web" ]; then
	echo "Rebuilding assistant-web"
	go build -o bin/assistant-web assistant/cmd/assistant-web
else
	echo "Rebuilding assistant and assistant-web"
	go build -o bin/assistant assistant/cmd/assistant
	go build -o bin/assistant-web assistant/cmd/assistant-web
fi

