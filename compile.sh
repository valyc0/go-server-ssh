#!/bin/bash

# Nome del file eseguibile
EXECUTABLE="ssh-server_go"

# Compilazione per Linux
echo "Compilazione per Linux..."
GOOS=linux GOARCH=amd64 go build -o "$EXECUTABLE-linux" ssh.go

# Compilazione per macOS
echo "Compilazione per macOS..."
GOOS=darwin GOARCH=amd64 go build -o "$EXECUTABLE-macos" ssh.go

# Compilazione per Windows
echo "Compilazione per Windows..."
GOOS=windows GOARCH=amd64 go build -o "$EXECUTABLE.exe" ssh.go

echo "Compilazione completata!"

