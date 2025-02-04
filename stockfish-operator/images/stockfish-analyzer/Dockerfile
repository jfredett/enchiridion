FROM ubuntu:24.04

RUN apt-get update && apt-get install -y wget

RUN wget https://github.com/official-stockfish/Stockfish/releases/latest/download/stockfish-ubuntu-x86-64-bmi2.tar

RUN tar -xvf stockfish-ubuntu-x86-64-bmi2.tar

RUN mv stockfish/stockfish-ubuntu-x86-64-bmi2 /usr/local/bin/stockfish

# Create an entrypoint script
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Set the entrypoint
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
