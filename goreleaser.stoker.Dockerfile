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
    && apt-get clean
ENTRYPOINT ["/usr/local/bin/stoker"]
COPY stoker /usr/local/bin/
