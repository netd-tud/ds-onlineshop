ARG BUILDPLATFORM=linux/amd64

FROM --platform=$BUILDPLATFORM debian:bookworm-slim
RUN apt-get update && \
    apt-get install -y curl && \
    rm -rf /var/lib/apt/lists/*

ENV TARGET="frontend-demo:8080"

# Copy load producer script
COPY ./dynamic-load-producer.sh /usr/local/bin/dynamic-load-producer.sh
RUN chmod u+x /usr/local/bin/dynamic-load-producer.sh

# Start load generation
CMD [ "/usr/local/bin/dynamic-load-producer.sh" ]
