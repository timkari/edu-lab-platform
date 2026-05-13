#!/bin/sh
# Кладёт ярлык Geany на рабочий стол (после того как базовый образ поднял LXDE / домашний каталог).
set -f

# В образе doro по умолчанию show_documents=0 — PCManFM не показывает файлы из ~/Desktop на рабочем столе.
DORO_DESKTOP_CONF=/usr/local/share/doro-lxde-wallpapers/desktop-items-0.conf
if test -f "$DORO_DESKTOP_CONF"; then
  sed -i 's/^show_documents=.*/show_documents=1/' "$DORO_DESKTOP_CONF" 2>/dev/null || true
fi

SRC=/usr/share/applications/geany-lab.desktop
if ! test -f "$SRC"; then
  SRC=/usr/share/applications/geany.desktop
fi
if ! test -f "$SRC"; then
  exit 0
fi

place() {
  home=$1
  owner=$2
  test -d "$home" || return 0
  grp=root
  if id "$owner" >/dev/null 2>&1; then
    grp=$(id -gn "$owner" 2>/dev/null || echo "$owner")
  else
    owner=root
  fi
  for desk in Desktop "Рабочий стол"; do
    d="$home/$desk"
    mkdir -p "$d"
    cp -f "$SRC" "$d/Geany.desktop"
    chmod 644 "$d/Geany.desktop"
    chown "$owner:$grp" "$d/Geany.desktop" 2>/dev/null || true
    chown "$owner:$grp" "$d" 2>/dev/null || true
  done
}

place /root root
for home in /home/*; do
  test -e "$home" || continue
  test -d "$home" || continue
  u=$(basename "$home")
  place "$home" "$u"
done

if test -d /etc/skel; then
  mkdir -p /etc/skel/Desktop
  cp -f "$SRC" /etc/skel/Desktop/Geany.desktop
  chmod 644 /etc/skel/Desktop/Geany.desktop
fi

update-desktop-database /usr/share/applications 2>/dev/null || true

# Подхватить новые иконки без перезапуска сессии
pids=$(pidof pcmanfm 2>/dev/null) || true
for p in $pids; do
  kill -HUP "$p" 2>/dev/null || true
done

exit 0
