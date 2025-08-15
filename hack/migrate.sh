#!/usr/bin/env sh

wget $1/steamapps/896660 \
    --header="Content-Type: application/json" \
    --post-data='{
      "apt_packages": [
        "ca-certificates"
      ],
      "launch_type": "server",
      "execs": [
        "rm -r /home/steam/docker /home/steam/docker_start_server.sh /home/steam/start_server_xterm.sh /home/steam/start_server.sh",
        "ln -s /home/steam/linux64/steamclient.so /usr/lib/x86_64-linux-gnu/steamclient.so"
      ],
      "entrypoint": ["/home/steam/valheim_server.x86_64"],
      "ports": [
        {
          "port": 2456
        }
      ]
    }' -O-

wget $1/steamapps/1963720 \
    --header="Content-Type: application/json" \
    --post-data='{
      "apt_packages": [
        "ca-certificates",
        "curl",
        "locales",
        "libxi6",
        "xvfb"
      ],
      "launch_type": "server",
      "execs": [
        "ln -s /home/steam/linux64/steamclient.so /usr/lib/x86_64-linux-gnu/steamclient.so"
      ],
      "entrypoint": [
        "/home/steam/_launch.sh",
        "-logfile",
        "/dev/stdout"
      ],
      "volumes": [
        {
          "path": "/home/steam/.config/unity3d/Pugstorm/Core Keeper/DedicatedServer"
        }
      ]
    }' -O-

wget $1/steamapps/2394010 \
    --header="Content-Type: application/json" \
    --post-data='{
      "apt_packages": [
        "ca-certificates",
        "xdg-user-dirs"
      ],
      "launch_type": "default",
      "ports": [
        {
          "port": 8211,
          "protocols": ["TCP", "UDP"]
        },
        {
          "port": 27015,
          "protocols": ["TCP", "UDP"]
        }
      ]
    }' -O-

wget $1/steamapps/1690800 \
    --header="Content-Type: application/json" \
    --post-data='{
      "launch_type": "default",
      "ports": [
        {
          "port": 7777,
          "protocols": ["TCP", "UDP"]
        },
        {
          "port": 8888,
          "protocols": ["TCP"]
        }
      ],
      "volumes": [
        {
          "path": "/home/steam/.config/Epic/FactoryGame/Saved/SaveGames"
        }
      ]
    }' -O-

wget $1/steamapps/2857200 \
    --header="Content-Type: application/json" \
    --post-data='{
      "launch_type": "default",
      "apt_packages": [
        "winehq-stable"
      ],
      "platform_type": "windows",
      "ports": [
        {
          "port": 7777,
          "protocols": ["TCP", "UDP"]
        },
        {
          "port": 27015,
          "protocols": ["TCP"]
        }
      ],
      "entrypoint": [
        "wine",
        "/home/steam/AbioticFactor/Binaries/Win64/AbioticFactorServer-Win64-Shipping.exe",
        "-log",
        "-newconsole",
        "-useperfthreads",
        "-NoAsyncLoadingThread"
      ]
    }' -O-
