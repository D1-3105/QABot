CURRENT_APP_VERSION := $(shell git describe --tags --long --always)

REGISTRY_USER := d13105
REGISTRY_PASSWORD ?=

DOCKER_IMAGE_NAME := ${REGISTRY_USER}/act-qabot:${CURRENT_APP_VERSION}

swagger_http_rest_api:
	swag init -g ./main.go -o ./docs


build_frontend:
	cd frontend/web-interface && yarn build

docker_final:
	docker build -t ${DOCKER_IMAGE_NAME} .

registry_login:
	echo "${REGISTRY_PASSWORD}" | docker login -u ${REGISTRY_USER} --password-stdin 2>/dev/null || true

push_image:
	docker push ${DOCKER_IMAGE_NAME}

upload_docker_artifacts: registry_login docker_final push_image

test:
	go test -v ./tests -args -logtostderr=true -v=1
