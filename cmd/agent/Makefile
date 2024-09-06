all: mrva-agent

MAI_TARGET := mrva-agent:0.1.24
mai: mrva-agent
mrva-agent:
	cd ../../ && docker build -t mrva-agent:0.1.24 -f cmd/agent/Dockerfile .
	touch $@

mai-serve: mai
	docker run --rm -it ${MAI_TARGET} /bin/bash

clean:
	-docker rmi -f ${MAI_TARGET}
	-rm mrva-agent

mai-push: mai
	docker tag ${MAI_TARGET} ghcr.io/hohn/${MAI_TARGET}
	docker push ghcr.io/hohn/${MAI_TARGET}
	touch $@


