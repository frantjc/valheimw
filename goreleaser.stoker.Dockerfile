FROM node:20.11.1-slim AS remix
WORKDIR /src/github.com/frantjc/sindri
COPY package.json yarn.lock ./
RUN yarn
COPY app/ app/
COPY public/ public/
COPY *.js *.ts tsconfig.json ./
RUN yarn build

FROM node:20.11.1-slim
RUN apt-get update -y \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        lib32gcc-s1 \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean \
    && groupadd --system stoker \
    && useradd --system --gid stoker --shell /bin/bash --create-home stoker
USER stoker
ENV NODE_ENV production
ENTRYPOINT ["/usr/local/bin/stoker"]
COPY stoker /usr/local/bin/
COPY server.js package.json /app/
COPY --from=remix /src/github.com/frantjc/sindri/build /app/build/
COPY --from=remix /src/github.com/frantjc/sindri/node_modules /app/node_modules/
