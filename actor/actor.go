package actor

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/BeforyDeath/raftik/store"
)

var (
	m         sync.Mutex
	redigoNil string = "redigo: nil returned"
)

type Actor struct {
	Store        *store.Store
	UUID         string
	Role         int
	Term         int
	VotingActive bool
	nodes        []string
	Max, Min     int
}

const (
	Follower = iota
	Candidate
	Leader
)

type voteRequest struct {
	From    string
	To      string
	Term    int
	Granted bool
}

type appendRequest struct {
	From  string
	To    string
	Term  int
	Msg   string
	Error bool
}

func (a *Actor) EraEnded() bool {
	m.Lock()
	defer m.Unlock()
	if !a.VotingActive && a.Role != Candidate {
		a.VotingActive = true
		fmt.Println("--- IM CANDIDATE ---")
		a.Role = Candidate
		a.Term++
		fmt.Println("TERM SET", a.Term)

		fmt.Println("SEND VOTING")
		err := a.setVoting()
		if err != nil {
			log.Fatal(err)
		}

		time.Sleep(time.Millisecond * time.Duration(a.Min/2))

		fmt.Println("GET VOTE")
		count, err := a.getVote()
		if err != nil {
			log.Fatal(err)
		}
		count++ //todo my vote

		nodes := len(a.nodes)
		fmt.Printf("VOTE:%v NODES:%v NIDE:%v\n", count, nodes, (nodes/2)+1)
		if nodes > 1 && count >= (nodes/2)+1 {
			fmt.Println("--- IM LEADER ---")
			a.Role = Leader
			a.VotingActive = false
			return false
		}
		fmt.Println("--- IM FOLLOWER ---")
		a.Role = Follower
		a.VotingActive = false
		return true
	}
	return true
}

func (a *Actor) setVoting() error {
	vr := voteRequest{
		From: a.UUID,
		Term: a.Term,
	}
	for _, uuid := range a.nodes {
		if uuid != a.UUID {
			vr.To = uuid
			vrj, err := json.Marshal(vr)
			if err != nil {
				return err
			}
			err = a.Store.SetVoteRequest(uuid, vrj)
			if err != nil {
				return err
			}
			fmt.Println("< VOTE\t", vr.To, a.Term)
		}
	}
	return nil
}

func (a *Actor) getVote() (int, error) {
	count := 0
	vote, err := a.Store.GetVoteReply(a.UUID)
	if err != nil {
		return 0, err
	}

	for _, v := range vote {
		res := new(voteRequest)
		err := json.Unmarshal([]byte(v), res)
		if err != nil {
			return 0, err
		}
		if a.Term < res.Term {
			a.Term = res.Term
			fmt.Println("TERM UP FROM VOTE", a.Term)
			return 0, nil
		}

		if res.Granted {
			count++
		}
		fmt.Println("> VOTE\t", res.From, res.Term, res.Granted)
	}
	return count, nil
}

func (a *Actor) CheckVoteRequest(eraReset func()) {
	ticker := time.NewTicker(time.Millisecond * time.Duration(a.Min))
	for range ticker.C {
		m.Lock()
		req, err := a.Store.GetVoteRequest(a.UUID)
		if err != nil && err.Error() != redigoNil {
			m.Unlock()
			log.Fatal(err)
			return
		}
		if err == nil {
			for _, v := range req {
				res := voteRequest{}
				err := json.Unmarshal([]byte(v), &res)
				if err != nil {
					m.Unlock()
					log.Fatal(err)
					return
				}
				fmt.Println("< REPLY\t", res.From, res.Term)

				reply := voteRequest{
					From: a.UUID,
					To:   res.From,
					Term: a.Term,
				}

				if a.Term >= res.Term {
					reply.Granted = false
				}

				if a.Term < res.Term {
					a.Term = res.Term
					reply.Term = res.Term
					fmt.Println("TERM UP FROM REPLY", a.Term)
					fmt.Println("--- IM FOLLOWER ---")
					a.Role = Follower
					eraReset()
					reply.Granted = true
				}

				replyj, err := json.Marshal(reply)
				if err != nil {
					m.Unlock()
					log.Fatal(err)
					return
				}

				err = a.Store.SetVoteReply(res.From, replyj)
				if err != nil {
					m.Unlock()
					log.Fatal(err)
					return
				}

				fmt.Println("> REPLY\t", reply.To, reply.Term, reply.Granted)
			}
		}
		m.Unlock()
	}
}

func (a *Actor) Leader() {
	fmt.Println("START LEADER SENDER")
	ticker := time.NewTicker(time.Millisecond * time.Duration(a.Min/2))
	for range ticker.C {
		m.Lock()
		if a.Role == Leader {
			ar := appendRequest{
				From: a.UUID,
				Term: a.Term,
			}
			for _, uuid := range a.nodes {
				if uuid != a.UUID {
					ar.To = uuid
					ar.Msg = fmt.Sprintf("Hello uuid (%v ...) %vnc", uuid[0:8], time.Now().Nanosecond())
					ar.Error = false

					if rand.Intn(100) <= 5 {
						ar.Msg = fmt.Sprintf("Leader set (%v ...) random error with a probability of 5 percent [%v]", a.UUID[0:8], time.Now())
						ar.Error = true
					}

					arj, err := json.Marshal(ar)
					if err != nil {
						m.Unlock()
						log.Fatal(err)
						return
					}
					err = a.Store.SetAppendRequest(uuid, arj)
					if err != nil {
						m.Unlock()
						log.Fatal(err)
						return
					}
					fmt.Println("> SEND\t", string(arj))
				}
			}
		}
		m.Unlock()
	}
}

func (a *Actor) Follower(eraReset func()) {
	fmt.Println("START FOLLOWER READER")
	ticker := time.NewTicker(time.Millisecond * time.Duration(a.Min/2))
	for range ticker.C {
		m.Lock()
		if a.Role != Leader {
			req, err := a.Store.GetAppendRequest(a.UUID)
			if err != nil && err.Error() != redigoNil {
				m.Unlock()
				log.Fatal(err)
				return
			}
			if err == nil {
				res := new(appendRequest)
				err := json.Unmarshal(req, res)
				if err != nil {
					m.Unlock()
					log.Fatal(err)
					return
				}
				if a.Role == Follower {
					if res.Error == true {
						a.Store.SetError(res.Msg + fmt.Sprintf(" get (%v ...)", a.UUID[0:8]))
					}

					fmt.Println("< READ\t FROM", res.From[0:8], "...", res.Term, res.Msg)
					eraReset()
				}

				if a.Role == Candidate {
					a.Role = Follower
					a.Term = res.Term
				}
			}
		}
		m.Unlock()
	}
}

func (a *Actor) Ping() {
	fmt.Println("START PING")
	ticker := time.NewTicker(time.Millisecond * time.Duration(a.Max))
	for range ticker.C {
		m.Lock()
		nodes, err := a.Store.GetNodes()
		if err != nil {
			m.Unlock()
			log.Fatal(err)
			return
		}
		a.nodes = nodes

		for _, uuid := range nodes {

			if uuid == a.UUID {
				err := a.Store.PingMe(a.UUID)
				if err != nil {
					m.Unlock()
					log.Fatal(err)
					return
				}
			}

			i, err := a.Store.Ping(uuid)
			if err != nil {
				m.Unlock()
				log.Fatal(err)
				return
			}

			if i > 10 {
				err := a.Store.Erase(uuid)
				if err != nil {
					m.Unlock()
					log.Fatal(err)
					return
				}
				fmt.Println("ERASE NODE UUID", uuid)
			}
		}
		m.Unlock()
	}
}
