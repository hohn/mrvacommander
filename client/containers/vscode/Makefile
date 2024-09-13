all: code-server-initialized

CSI_TARGET := code-server-initialized:0.1.24
csi: mk.code-server-initialized
mk.code-server-initialized:
	docker build -t ${CSI_TARGET} .
	touch $@

csi-serve: csi
	docker run -d -p 9080:9080 ${CSI_TARGET}

clean:
	-docker rmi -f ${CSI_TARGET}
	-rm mk.code-server-initialized

# Targets below are used after some manual setup of the container.  See README.org
# for details

csi-push: mk.csi-push
mk.csi-push: csi
	docker tag ${CSI_TARGET} ghcr.io/hohn/${CSI_TARGET}
	docker push ghcr.io/hohn/${CSI_TARGET}
	touch $@

csi-test:
	docker pull ghcr.io/hohn/${CSI_TARGET}
	docker run --rm -d -p 9080:9080 --name test-code-server-codeql\
		ghcr.io/hohn/${CSI_TARGET}
