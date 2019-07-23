ifeq ($(REGISTRY),)
	REGISTRY = quay.io/minio/
endif
ifeq ($(VERSION),)
	VERSION = latest
endif
IMAGE = $(REGISTRY)minio-provisioner:$(VERSION)
MUTABLE_IMAGE = $(REGISTRY)minio-provisioner:latest

all build:
	GOOS=linux go build ./cmd/minio-provisioner
.PHONY: all build

container: build quick-container
.PHONY: container

quick-container:
	cp minio-provisioner deploy/docker
	docker build -t $(MUTABLE_IMAGE) deploy/docker
	docker tag $(MUTABLE_IMAGE) $(IMAGE)
.PHONY: quick-container

push: container
	docker push $(IMAGE)
	docker push $(MUTABLE_IMAGE)
.PHONY: push

test-all: test test-e2e

test:
	go test `go list ./... | grep -v 'vendor\|test\|demo'`
.PHONY: test

test-e2e:
	cd ./test/e2e; ./test.sh
.PHONY: test-e2e

clean:
	rm -f minio-provisioner
	rm -f deploy/docker/minio-provisioner
	rm -rf test/e2e/vendor
.PHONY: clean
