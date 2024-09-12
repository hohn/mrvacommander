# Overview

TODO diagram

TODO Style notes
- NO package init() functions
- Dynamic behaviour must be explicit
 

## Client CodeQL Database Selector
Separate from the server's downloading of databases, a client-side interface is needed to generate the `databases.json` file. This

1.  must be usable from the shell
2.  must be interactive (Python, Jupyter)
3.  is session based to allow iterations on selection / narrowing
4.  must be queryable. There is no need to reinvent sql / dataframes

Python with dataframes is ideal for this; the project is in `client/`.

## Reverse proxy
For testing, replay flows using mitmweb.  This is faster and simpler than using
gh-mrva or the VS Code plugin.

-   Set up the virtual environment and install tools

        python3.11 -m venv venv
        source venv/bin/activate
        pip install mitmproxy

For intercepting requests:

1.  Start mitmproxy to listen on port 8080 and forward requests to port 8081, with
    web interface

        mitmweb --mode reverse:http://localhost:8081 -p 8080

1.  Change `server` ports in `docker-compose.yml` to 

        ports:
        - "8081:8080" # host:container

1.  Start the containers.

1.  Submit requests.

3.  Save the flows for later replay.

One such session is in `tools/mitmweb-flows`; it can be loaded to replay the
requests:

1.  start `mitmweb --mode reverse:http://localhost:8081 -p 8080`
2.  `file` > `open` > `tools/mitmweb-flows`
3.  replay at least the submit, status, and download requests

## Cross-compile server on host, run it in container 
These are simple steps using a single container.

1.  build server on host

        GOOS=linux GOARCH=arm64 go build

2.  build docker image

        cd cmd/server
        docker build -t server-image .

3.  Start container with shared directory

    ```sh
    docker run -it \
           -v   /Users/hohn/work-gh/mrva/mrvacommander:/mrva/mrvacommander \
           server-image
    ```

4.  Run server in container

        cd /mrva/mrvacommander/cmd/server/ && ./server

## Using docker-compose
### Steps to build and run the server

Steps to build and run the server in a multi-container environment set up by
docker-compose. 

1.  Built the server-image, above

1.  Build server on host

        cd ~/work-gh/mrva/mrvacommander/cmd/server/
        GOOS=linux GOARCH=arm64 go build

1.  Start the containers

        cd ~/work-gh/mrva/mrvacommander/
        docker-compose down
        docker-compose up -d
    
4.  Run server in its container

        cd ~/work-gh/mrva/mrvacommander/
        docker exec -it server bash
        cd /mrva/mrvacommander/cmd/server/ 
        ./server -loglevel=debug -mode=container

1.  Test server from the host via

        cd ~/work-gh/mrva/mrvacommander/tools
        sh ./request_16-Jun-2024_11-33-16.curl

1.  Follow server logging via

        cd ~/work-gh/mrva/mrvacommander
        docker-compose up -d
        docker-compose logs -f server

1.  Completely rebuild all containers.  Useful when running into docker errors

        cd ~/work-gh/mrva/mrvacommander
        docker-compose up --build

1.  Start the server containers and the desktop/demo containers

        cd ~/work-gh/mrva/mrvacommander/
        docker-compose down --remove-orphans
        docker-compose -f docker-compose-demo.yml up -d

1.  Test server via remote client by following the steps in [gh-mrva](https://github.com/hohn/gh-mrva/blob/connection-redirect/README.org#compacted-edit-run-debug-cycle)

### Some general docker-compose commands

2.  Get service status

        docker-compose ps
        
3.  Stop services

        docker-compose down
        
4.  View all logs

        docker-compose logs

5.  check containers from server container

        docker exec -it server bash
        curl -I http://rabbitmq:15672

### Use the minio ql database db

1.  Web access via

        open http://localhost:9001/login

    username / password are in `docker-compose.yml` for now.  The ql db listing 
    will be at

        http://localhost:9001/browser/qldb

1.  Populate the database by running

        ./populate-dbstore.sh
        
    from the host.

1.  The names in the bucket use the `owner_repo` format for now,
    e.g. `google_flatbuffers_db.zip`.
    TODO This will be enhanced to include other data later

1.  Test Go's access to the dbstore -- from the host -- via

        cd ./test
        go test -v

    This should produce

        === RUN   TestDBListing
        dbstore_test.go:44: Object Key: google_flatbuffers_db.zip
        dbstore_test.go:44: Object Key: psycopg_psycopg2_db.zip

### Use the minio query pack db

1.  Web access via

        open http://localhost:19001/login

    username / password are in `docker-compose.yml` for now.  The ql db listing 
    will be at

        http://localhost:19001/browser/qpstore

    
### To run Use the minio query pack db
