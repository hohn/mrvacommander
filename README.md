# Overview

TODO diagram

TODO Style notes
- NO package init() functions
- Dynamic behaviour must be explicit
 
## Cross-compile server on host, run it in container 
These are simple steps using a single container.

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

## Using docker-compose
Steps to build and run the server in a multi-container environment set up by docker-compose.

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

1.  Test server via remote client by following the steps in [gh-mrva](https://github.com/hohn/gh-mrva/blob/connection-redirect/README.org#compacted-edit-run-debug-cycle)
    



Some general docker-compose commands

2.  Get service status

        docker-compose ps
        
3.  Stop services

        docker-compose down
        
4.  View all logs

        docker-compose logs

5.  check containers from server container

        docker exec -it server bash
        curl -I postgres:5432
        curl -I http://rabbitmq:15672


Some postgres specific commands

1.  Access PostgreSQL
    
        psql -h localhost -p 5432 -U exampleuser -d exampledb

1.  List all tables
    
        \dt

To run pgmin, the minimal go/postgres test part of this repository:

1.  Run pgmin

    ```sh
    cd ~/work-gh/mrva/mrvacommander/cmd/postgres
    GOOS=linux GOARCH=arm64 go build
    docker exec -it server bash
    /mrva/mrvacommander/cmd/postgres/postgres
    ```

    Exit the container.  Back on the host:
    
        psql -h localhost -p 5432 -U exampleuser -d exampledb
        \dt
    
    Should show

                List of relations
         Schema |    Name     | Type  |    Owner
        --------+-------------+-------+-------------
         public | owner_repos | table | exampleuser    


1.  Check table contents

        exampledb=# select * from owner_repos;
         owner |  repo
        -------+---------
         foo   | foo/bar
