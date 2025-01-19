# corekeeper

A Dockerfile to build a container image for the Core Keeper server. Relies on [`boiler`](../docs/boiler.md) to provide the base of the container image.

## build

First, ensure an instance of `boiler` is running. This can be done most easily by executing the following in this directory, but can be achieved through a number of different methods if one was so inclined.

```sh
docker compose up --build -d
```

> If your instance of `boiler` is running somewhere other than `localhost:5000` (for example, because you did not follow the previous step verbatim), modify the `FROM` directive in [`Dockerfile`](Dockerfile) to reference it.

Next, build the Core Keeper server image via the following command in this directory.

```sh
docker build .
```
