#!/bin/sh
# Базовый образ dorowu пересоздаёт /root при старте — ярлыки ставим с задержкой после /startup.sh.
( sleep 3;  /usr/local/bin/install-lab-desktop-icons.sh ) &
( sleep 10; /usr/local/bin/install-lab-desktop-icons.sh ) &
( sleep 25; /usr/local/bin/install-lab-desktop-icons.sh ) &
( sleep 60; /usr/local/bin/install-lab-desktop-icons.sh ) &
exec /startup-orig.sh "$@"
