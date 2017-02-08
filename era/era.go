package era

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type Era struct {
	sync.Mutex
	timer   *time.Timer
	exitCh  chan bool
	timeout time.Duration
	min     int
	max     int
	outFunc func() bool
	active  bool
}

func NewEra(min, max int, f func() bool) Era {
	rand.Seed(time.Now().UnixNano())
	e := Era{
		min:     min,
		max:     max,
		exitCh:  make(chan bool, 1),
		outFunc: f,
	}
	e.newDuration()
	return e
}

func (e *Era) Start() {
	e.Lock()
	if e.timer == nil {
		e.timer = time.NewTimer(e.timeout)
	}
	e.active = true
	e.Unlock()
	go e.start()
	fmt.Println("ERA START")
}

func (e *Era) start() {
	for {
		select {
		case <-e.timer.C:
			e.hasEnded()
		case <-e.exitCh:
			return
		}
	}
}

func (e *Era) Reset() {
	e.Lock()
	defer e.Unlock()

	if e.timer == nil {
		e.timer = time.NewTimer(e.timeout)
	}

	if !e.timer.Stop() {
		select {
		case <-e.timer.C:
		default:
		}
	}

	e.newDuration()
	e.timer.Reset(e.timeout)
	if !e.active {
		go e.start()
	}

	fmt.Println("ERA RESET")
}

func (e *Era) Stop() {
	e.timer.Stop()
	e.exitCh <- true
	e.Lock()
	e.active = false
	e.Unlock()
	fmt.Println("ERA STOP")
}

func (e *Era) newDuration() {
	e.timeout = time.Duration(rand.Intn(e.max-e.min+1)+e.min) * time.Millisecond
}

func (e *Era) ResetDuring(min, max int) {
	e.Lock()
	e.min = min
	e.max = max
	e.Unlock()
	e.Reset()
}

func (e *Era) hasEnded() {
	e.Lock()
	fmt.Printf("ERA HAS ENDED (%v)\n", e.timeout)
	e.Unlock()

	if e.outFunc() {
		e.Reset()
	} else {
		e.Stop()
	}
}
