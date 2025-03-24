#!/bin/sh

# Start the application based on SUBCOMMAND
case "$SUBCOMMAND" in
    "web")
        echo "Starting web server..."
        ls -la
        ./expensetrace web
        ;;
    "tui")
        echo "Starting TUI interface..."
        ls -la
        ./expensetrace tui
        ;;
    *)
        echo "Unknown mode: $SUBCOMMAND"
        echo "Available modes: web, tui"
        exit 1
        ;;
esac
