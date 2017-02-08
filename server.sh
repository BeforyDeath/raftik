#!/bin/bash

BASEDIR=`dirname $0`
PROJECT_PATH=`cd $BASEDIR; pwd`

start(){
    docker run -d \
            --hostname redis \
            --name redis \
            -p 6379:6379 \
            -v $PROJECT_PATH/data/redis:/data \
            redis
    docker ps
}

stop(){
    docker stop $(docker ps -a -q)
}

clear(){
    docker rm $(docker ps -a -q)
}

case "$1" in
        start)
            start
            ;;

        stop)
            stop
            clear
            ;;

        restart)
            stop
            start
            ;;

        clear)
            clear
            ;;

        influx)
            influx
            ;;

        *)
            echo $"Usage: $0 {start|stop|restart|clear|influx}"
            exit 1
esac
