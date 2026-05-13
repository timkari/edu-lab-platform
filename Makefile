# Образ лаборатории: Geany на рабочем столе noVNC (linux/amd64 — как при запуске с macOS).
LAB_IMAGE_TAG ?= edu-lab-lab-vnc:geany

.PHONY: lab-image up
lab-image:
	docker build --platform linux/amd64 -t $(LAB_IMAGE_TAG) -f docker/lab-vnc/Dockerfile docker/lab-vnc

# Собрать образ лаборатории и поднять платформу с LAB_IMAGE (шаблон в БД обновится при старте).
up: lab-image
	LAB_IMAGE=$(LAB_IMAGE_TAG) docker compose up -d --build
