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

## Using docker
1.  start the services

        docker-compose up -d

    
2.  get status

        docker-compose ps
        
3.  stop services

        docker-compose down
        
4.  view all logs

        docker-compose logs

5.  check containers from server container

        docker exec -it server bash
        curl -I postgres:5432
        curl -I http://rabbitmq:15672
        
1.  Accessing PostgreSQL
    
        psql -h localhost -p 5432 -U exampleuser -d exampledb

1.  List all tables
    
        \dt

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
