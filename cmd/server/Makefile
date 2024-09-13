all: mrva-server

MSI_TARGET := mrva-server:0.1.24
msi: mk.mrva-server
mrva-server: mk.mrva-server
mk.mrva-server:
	cd ../../ && docker build -t mrva-server:0.1.24 -f cmd/server/Dockerfile .
	touch $@

msi-serve: msi
	docker run --rm -it ${MSI_TARGET} /bin/bash

clean:
	-docker rmi -f ${MSI_TARGET}
	-rm mrva-server

msi-push: mk.msi-push
mk.msi-push: mk.mrva-server
	docker tag ${MSI_TARGET} ghcr.io/hohn/${MSI_TARGET}
	docker push ghcr.io/hohn/${MSI_TARGET}
	touch $@

msi-test:
	docker pull ghcr.io/hohn/${MSI_TARGET}
	docker run --rm -it --name test-mrva-server-codeql ghcr.io/hohn/${MSI_TARGET} sh

