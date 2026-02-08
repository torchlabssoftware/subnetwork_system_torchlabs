#!/bin/sh
set -e

# Generate certificates if they don't exist
if [ ! -f /proxy.crt ] || [ ! -f /proxy.key ]; then
    echo "Generating SSL certificates..."
    /proxy keygen
fi

# If the first argument is a flag or known subcommand, run proxy
case "$1" in
    -*)
        set -- /proxy "$@"
        ;;
    http|socks|tserver|tclient|tbridge|keygen)
        set -- /proxy "$@"
        ;;
esac

# Execute the passed command
exec "$@"
