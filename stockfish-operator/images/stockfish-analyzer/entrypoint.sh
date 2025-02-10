#!/usr/bin/env bash

# Define the socket path for Stockfish
SOCKET_PATH=/tmp/stockfish-uci.socket

# Start Stockfish in the background, redirecting its input/output to the socket
mkfifo "$SOCKET_PATH"
stockfish < "$SOCKET_PATH" &> /dev/null &
STOCKFISH_PID=$!

# Set debug file
echo "setoption name Debug Log File value /tmp/stockfish_output.txt" > "$SOCKET_PATH"

echo "Running analysis"
# Use positional arguments as UCI commands
for COMMAND in "$@"; do
    # Send command to Stockfish through the socket and also write it to the output
    echo "> $COMMAND"
    echo "$COMMAND" > "$SOCKET_PATH"
    sleep 10
done


# Wait for Stockfish to finish processing all commands
wait $STOCKFISH_PID &> /dev/null

# Capture all output from Stockfish
OUTPUT=$(cat /tmp/stockfish_output.txt)

echo "Sending output"
echo "$OUTPUT"

# Send the captured output to Redis under the key 'analysis'
# HACK: Skipping this for now
#   redis-cli -h $REDIS_URL set "stockfish/$UUID/analysis.log" "$OUTPUT"

# Clean up, not super necessary since the container will die, but stewardship.
rm -f "$SOCKET_PATH"
rm -f /tmp/stockfish_output.txt
