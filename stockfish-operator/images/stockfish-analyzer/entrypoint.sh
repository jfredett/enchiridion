#!/usr/bin/env bash

for arg in "$@" ; do
    echo "$arg"
done

# Grab the uci script however it has been provided to us
# Write it to a file
# Invoke Stockfish with the input, output to stdout (controller will read our log file)
# Exit.

