GROUP_NAME:=platform
PROJECT_NAME:=arc-storage
API_ROOT=/api/data/v1/history

.PHONY: build image run swagger-client swagger-server

build:
	bash build.sh ${PROJECT_NAME} ./${PROJECT_NAME}

image:
	docker pull golang:1.20.8-bullseye
	docker build -t ${GROUP_NAME}/${PROJECT_NAME}:dev .

run:
	docker run --rm \
	-p 80:80 \
	-e BASIC_INSWARM=false \
	-e TAOS_TAOSENABLE=false \
	${GROUP_NAME}/${PROJECT_NAME}:dev

swagger-json:
	rm -f ./swagger.json
	wget http://127.0.0.1/api${API_ROOT}/swagger/swagger.json

swagger-client:
	docker pull golang:1.20.8-bullseye
	docker run --rm -v `pwd`:/src quay.io/goswagger/swagger:v0.28.0 \
	generate client -t /src -f /src/swagger.json --skip-validation \
	--client-package=api/client --model-package=api/models

swagger-server:
	docker pull quay.io/goswagger/swagger:v0.28.0
	docker run --rm -v `pwd`:/src quay.io/goswagger/swagger:v0.28.0 \
	generate server -t /src -f /src/swagger.json --skip-validation \
	--server-package=api/server --model-package=api/models --main-package=test-server
