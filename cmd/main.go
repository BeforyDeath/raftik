package main

import (
	"fmt"
	"os"
	"time"

	"github.com/BeforyDeath/raftik/actor"
	"github.com/BeforyDeath/raftik/era"
	"github.com/BeforyDeath/raftik/store"
	log "github.com/Sirupsen/logrus"
	uuid "github.com/nu7hatch/gouuid"
)

var (
	min int = 100
	max int = 200
	//min int = 1000 // send 500ms
	//max int = 1500
)

func main() {
	redis, err := store.NewStore("tcp", "localhost:6379")
	if err != nil {
		log.Error(err)
	}
	defer redis.Close()

	UUID, err := uuid.NewV4()
	if err != nil {
		log.Fatal(err)
	}

	Actor := actor.Actor{
		Store: &redis,
		UUID:  UUID.String(),
		Role:  actor.Follower,
		Term:  1,
		Min:   min,
		Max:   max,
	}

	redis.AddUUID(Actor.UUID)

	fmt.Println("START UUID", Actor.UUID)

	fmt.Println("NEW ERA ", min, max)
	Era := era.NewEra(min, max, Actor.EraEnded)

	go Actor.CheckVoteRequest(Era.Reset)
	go Actor.Leader()
	go Actor.Follower(Era.Reset)
	go Actor.Ping()

	//todo время на пинг, удаление данных о мёртвых нодах
	time.Sleep(time.Second * 5)
	Era.Start()

	time.Sleep(time.Minute * 60)

	os.Exit(0)

}
