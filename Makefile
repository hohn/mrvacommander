all: server agent

.phony: view

view: README.html
	open $<

html: README.html

%.html: %.md
	pandoc --toc=true --standalone $< --out $@

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
