build:
	cd cmd/server && GOOS=linux GOARCH=arm64 go build

fullbuild:
	cd cmd/server && GOOS=linux GOARCH=arm64 go build -a

sendsubmit:
	cd tools && sh ./submit-request.curl

lint:
	golangci-lint run cmd/... pkg/...
