IMAGE_TAG ?= quay.io/myeung/vault-go-client:v0.0.1

.PHONY: binary
binary:
	go build -o vault-go-client ./cmd/.

.PHONY: run-binary
run-binary: binary
	./vault-go-client

.PHONY: docker-build
docker-build:
	docker build -t ${IMAGE_TAG} .

.PHONY: docker-push
docker-push:
	docker push ${IMAGE_TAG}
