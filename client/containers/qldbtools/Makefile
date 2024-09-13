DBT_TARGET := client-qldbtools-container:0.1.24

# Build the qldbtools container image
dbt: mk.client-qldbtools-container
mk.client-qldbtools-container:
	docker build -t ${DBT_TARGET} .
	touch $@

# Run a shell in the container with the qldbtools
dbt-run: dbt
	docker run --rm -it ${DBT_TARGET} /bin/bash

# Run one of the scripts in the container as check.  Should exit with error. 
dbt-check: dbt
	docker run --rm -it ${DBT_TARGET} mc-db-initial-info

dbt-push: mk.dbt-push
mk.dbt-push: dbt
	docker tag ${DBT_TARGET} ghcr.io/hohn/${DBT_TARGET} 
	docker push ghcr.io/hohn/${DBT_TARGET}
	touch $@

dbt-test:
	docker pull ghcr.io/hohn/${DBT_TARGET}
	docker run --rm -it --name test-dbt-server ghcr.io/hohn/${DBT_TARGET} sh
