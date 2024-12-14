# sindri

Sindri is a toolkit for turning Steamapps into containers. This repository also houses tools built from this toolkit: `valheimw`, `mist`, `boiler` and `boil`.

## valheimw

## boiler

`boiler` is a read-only container registry for pulling images with Steam apps installed on them. The base of the images is `debian:stable-slim`. Images are non-root and `steamcmd` is never installed on them, so there's no leftover files from it on the filesystem or in the layers.

## boil

## mist
