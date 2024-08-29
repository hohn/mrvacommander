all: image

image:
	docker build -t mrvadata .

image-shell: image
	docker run --rm -it mrvadata sh

image-push: image
	docker tag mrvadata ghcr.io/hohn/mrvadata:0.1.24
	docker push ghcr.io/hohn/mrvadata:0.1.24

image-test:
	docker pull ghcr.io/hohn/mrvadata:0.1.24
	docker run --rm -it --name test-mrvadata ghcr.io/hohn/mrvadata:0.1.24 sh

