FROM debian:stable-slim
RUN apt-get update -y \
    && apt-get install -y --no-install-recommends \
        x11-apps \
        xauth \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean
ENTRYPOINT ["xeyes"]
