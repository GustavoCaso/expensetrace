#!/bin/sh

# Start the application based on EXPENSETRACE_SUBCOMMAND
case "$EXPENSETRACE_SUBCOMMAND" in
    "web")
        echo "Starting web server..."
        ./expensetrace web
        ;;
    "tui")
        echo "Starting TUI interface..."
        ./expensetrace tui
        ;;
    *)
        echo "Unknown mode: $EXPENSETRACE_SUBCOMMAND"
        echo "Available modes: web, tui"
        exit 1
        ;;
esac
