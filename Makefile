all: server agent

.phony: view

view: README.html
	open $<

html: README.html

%.html: %.md
	pandoc --toc=true --standalone $< --out $@

# Build the qldbtools container image
dbt: client-qldbtools-container
client-qldbtools-container:
	cd client/containers/qldbtools && \
		docker build -t $@:0.1.24 .
	touch $@

# Run a shell in the container with the qldbtools
dbt-run: dbt
	docker run --rm -it client-qldbtools-container:0.1.24 /bin/bash

# Run one of the scripts in the container as check
dbt-check: dbt
	docker run --rm -it client-qldbtools-container:0.1.24 mc-db-initial-info

dbt-push: dbt
	docker tag client-qldbtools-container:0.1.24 ghcr.io/hohn/client-qldbtools-container:0.1.24 
	docker push ghcr.io/hohn/client-qldbtools-container:0.1.24
	touch $@


ghm: client-ghmrva-container
client-ghmrva-container:
	cd client/containers/ghmrva && \
		docker build -t $@:0.1.24 .
	touch $@

ghm-push: ghm
	docker tag client-ghmrva-container:0.1.24 ghcr.io/hohn/client-ghmrva-container:0.1.24 
	docker push ghcr.io/hohn/client-ghmrva-container:0.1.24 
	touch $@

ghm-run:
	docker run --rm client-ghmrva-container --help


server:
	cd cmd/server && GOOS=linux GOARCH=arm64 go build

agent:
	cd cmd/agent && GOOS=linux GOARCH=arm64 go build

fullbuild:
	cd cmd/server && GOOS=linux GOARCH=arm64 go build -a

sendsubmit:
	cd tools && sh ./submit-request.curl

# Requires
#		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
lint:
	golangci-lint run cmd/... pkg/...

deps:
	godepgraph -maxlevel 4 -nostdlib  -i github.com/minio/minio-go ./cmd/server | dot -Tpdf > deps-server.pdf && open deps-server.pdf

depa:
	godepgraph -maxlevel 4 -nostdlib  -i github.com/minio/minio-go ./cmd/agent | dot -Tpdf > deps-agent.pdf && open deps-agent.pdf
