ghm: mk.client-ghmrva-container
mk.client-ghmrva-container:
	docker build -t client-ghmrva-container:0.1.24 .
	touch $@

ghm-push: mk.ghm-push
mk.ghm-push: mk.client-ghmrva-container
	docker tag client-ghmrva-container:0.1.24 ghcr.io/hohn/client-ghmrva-container:0.1.24 
	docker push ghcr.io/hohn/client-ghmrva-container:0.1.24 
	touch $@

ghm-run:
	docker run --rm -it ghcr.io/hohn/client-ghmrva-container:0.1.24 /bin/bash
