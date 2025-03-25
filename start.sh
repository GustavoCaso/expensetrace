#!/bin/sh

# Start the application based on SUBCOMMAND
case "$SUBCOMMAND" in
    "web")
        echo "Starting web server..."
        ./expensetrace web
        ;;
    "tui")
        echo "Starting TUI interface..."
        ./expensetrace tui
        ;;
    *)
        echo "Unknown mode: $SUBCOMMAND"
        echo "Available modes: web, tui"
        exit 1
        ;;
esac
