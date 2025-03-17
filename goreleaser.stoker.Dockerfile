FROM node:20.11.1-slim AS stoker
RUN apt-get update -y \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        lib32gcc-s1 \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean
COPY stoker /usr/local/bin/
ENTRYPOINT ["stoker"]
