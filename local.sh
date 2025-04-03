#!/bin/bash

# Configuration
# First check environment variable, otherwise use default path
DB_PATH=${EXPENSETRACE_DB:-expenses.db}
BACKUP_DIR="./db_backups"

# Create backup directory if it doesn't exist
mkdir -p "$BACKUP_DIR"

# Create backup with timestamp
if [ -f $DB_PATH ]; then
    TIMESTAMP=$(date +%Y%m%d_%H%M%S)
    BACKUP_FILENAME=$(basename $DB_PATH).backup.$TIMESTAMP
    BACKUP_PATH=$BACKUP_DIR/$BACKUP_FILENAME
    
    cp $DB_PATH $BACKUP_PATH
    echo "Created backup at $BACKUP_PATH"
else
    echo "Warning: Database file $DB_PATH not found, no backup created"
fi

# Start the application
go run main.go "$@"
