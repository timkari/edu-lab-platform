FROM dorowu/ubuntu-desktop-lxde-vnc:latest

USER root

ENV DEBIAN_FRONTEND=noninteractive \
    DONT_PROMPT_WSL_INSTALL=1

# Base image may include third-party apt repos (e.g. Chrome) without keys.
# We don't need them; remove to keep builds reproducible.
RUN rm -f /etc/apt/sources.list.d/google-chrome.list /etc/apt/sources.list.d/google-chrome*.list || true

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    gpg \
    apt-transport-https \
    && rm -rf /var/lib/apt/lists/*

# Install Visual Studio Code (GUI)
RUN mkdir -p /etc/apt/keyrings \
    && curl -fsSL https://packages.microsoft.com/keys/microsoft.asc | gpg --dearmor > /etc/apt/keyrings/microsoft.gpg \
    && chmod go+r /etc/apt/keyrings/microsoft.gpg \
    && echo "deb [arch=amd64 signed-by=/etc/apt/keyrings/microsoft.gpg] https://packages.microsoft.com/repos/code stable main" > /etc/apt/sources.list.d/vscode.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends code \
    && rm -rf /var/lib/apt/lists/*

# Desktop shortcut on the main screen (Desktop)
# Make Desktop point to the persistent workdir (/home/ubuntu/work)
RUN mkdir -p /home/ubuntu/work \
    && rm -rf /root/Desktop \
    && ln -s /home/ubuntu/work /root/Desktop \
    && printf '%s\n' \
      '[Desktop Entry]' \
      'Type=Application' \
      'Name=VS Code' \
      'Comment=Visual Studio Code' \
      'Exec=env DONT_PROMPT_WSL_INSTALL=1 code --no-sandbox --user-data-dir=/root/.vscode-root' \
      'Icon=code' \
      'Terminal=false' \
      'Categories=Development;IDE;' \
      > /root/Desktop/VSCode.desktop \
    && chmod +x /root/Desktop/VSCode.desktop

