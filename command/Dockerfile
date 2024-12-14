FROM debian:stable-slim
# RUN apt-get update -y \
#     && apt-get install -y --no-install-recommends \
#         sudo \
#     && rm -rf /var/lib/apt/lists/*
RUN groupadd -r boil
RUN useradd -r -g boil -m -d /home/boil -s /bin/bash boil
# RUN echo "boil ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers
USER boil
