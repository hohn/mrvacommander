# Overview

TODO diagram

TODO Style notes
- NO package init() functions
- Dynamic behaviour must be explicit
 
## cross-compile server on host, run it in container 
1.  build server on host

        GOOS=linux GOARCH=arm64 go build

2.  build docker image

        docker build -t server-image .

3.  Start container with shared directory

    ```sh
    docker run -it \
           -v   /Users/hohn/work-gh/mrva/mrvacommander:/mrva/mrvacommander \
           server-image
    ```

4.  Run server in container

        cd /mrva/mrvacommander/cmd/server/ && ./server
