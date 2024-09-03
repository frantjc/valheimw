FROM steamcmd/steamcmd:debian-12
RUN apt-get update -y \
    && apt-get install -y --no-install-recommends \
        xvfb \
        libxi6 \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /
ENTRYPOINT ["/ladon"]
COPY ladon /
