FROM alpine:latest

RUN apk add --no-cache stockfish

# Create an entrypoint script
COPY stockfish-entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Set the entrypoint
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
