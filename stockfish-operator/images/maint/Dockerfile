# A docker container w/ the redis client on it

FROM ubuntu:24.04

RUN apt-get update && apt-get install -y wget redis

# Create an entrypoint script
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Set the entrypoint
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]

