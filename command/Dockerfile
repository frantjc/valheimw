FROM debian:stable-slim
RUN groupadd --system boil \
    && useradd --system --gid boil --shell /bin/bash --create-home boil
USER boil
