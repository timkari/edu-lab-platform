# Рабочий стол noVNC: Geany. Ярлык кладётся при старте контейнера (entrypoint), т.к. /startup.sh
# в базовом образе dorowu пересоздаёт домашний каталог и затирает файлы, созданные при build.
# Сборка: docker build --platform linux/amd64 -t edu-lab-lab-vnc:geany -f Dockerfile .
FROM dorowu/ubuntu-desktop-lxde-vnc

USER root
ENV DEBIAN_FRONTEND=noninteractive

# Репозиторий Google Chrome в базе без ключа — убираем.
RUN find /etc/apt/sources.list.d -maxdepth 1 -type f \( -iname '*google*' -o -iname '*chrome*' \) -delete 2>/dev/null || true \
  && (grep -rl 'dl\.google\.com' /etc/apt/sources.list.d 2>/dev/null | while read -r f; do rm -f "$f"; done) || true \
  && sed -i '/dl\.google\.com/d' /etc/apt/sources.list 2>/dev/null || true \
  && apt-get update \
  && apt-get install -y --no-install-recommends geany \
  && rm -rf /var/lib/apt/lists/*

RUN cat > /usr/share/applications/geany-lab.desktop <<'DESK'
[Desktop Entry]
Version=1.0
Type=Application
Name=Geany
Name[ru]=Geany — редактор кода
Comment=Бесплатный редактор кода
Exec=geany %F
Icon=geany
Terminal=false
Categories=Development;TextEditor;
StartupNotify=false
Keywords=code;editor;IDE;
DESK

RUN chmod 644 /usr/share/applications/geany-lab.desktop

# Иначе PCManFM не отображает содержимое ~/Desktop на рабочем столе (doro: show_documents=0).
RUN test -f /usr/local/share/doro-lxde-wallpapers/desktop-items-0.conf \
  && sed -i 's/^show_documents=.*/show_documents=1/' /usr/local/share/doro-lxde-wallpapers/desktop-items-0.conf

COPY install-lab-desktop-icons.sh /usr/local/bin/install-lab-desktop-icons.sh
COPY entrypoint.sh /usr/local/bin/edu-lab-vnc-entrypoint.sh
RUN chmod 755 /usr/local/bin/install-lab-desktop-icons.sh /usr/local/bin/edu-lab-vnc-entrypoint.sh \
  && mv /startup.sh /startup-orig.sh

ENTRYPOINT ["/usr/local/bin/edu-lab-vnc-entrypoint.sh"]
