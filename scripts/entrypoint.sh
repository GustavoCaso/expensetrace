#!/bin/sh
set -e

# Default PUID and PGID to 1000 if not set
PUID=${PUID:-1000}
PGID=${PGID:-1000}

echo "Starting with PUID=$PUID and PGID=$PGID"

# Only modify if different from current values
if [ "$(id -u expensetrace)" != "$PUID" ] || [ "$(id -g expensetrace)" != "$PGID" ]; then
    echo "Adjusting user and group IDs..."
    groupmod -g "$PGID" expensetrace
    usermod -u "$PUID" expensetrace
    echo "User and group IDs adjusted successfully"
fi

# Fix ownership of /data directory
echo "Fixing ownership of /data directory..."
chown -R "$PUID:$PGID" /data

# Ensure binary is executable
chmod +x /app/expensetrace

# Drop privileges and execute application
echo "Starting expensetrace as user expensetrace (UID=$PUID, GID=$PGID)"
exec su-exec expensetrace /app/expensetrace "$@"
