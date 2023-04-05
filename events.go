package minevents

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"io/ioutil"
	"sync"
)

// 支持流控跟本地恢复策略

const (
	ConcurrentNum = 3
)

type Manager struct {
	functions      map[EventType][]FuncCallback
	eventChain     chan Event
	L              *sync.Mutex
	Limiter        *ConnLimiter
	wg             *sync.WaitGroup
	sendEventMutex *sync.Mutex
	unExecuted     []Event     // new slice to store unExecuted events
	unExecutedL    *sync.Mutex // new mutex to protect unExecuted slice
}

var instance *Manager
var once sync.Once
var done chan bool

const (
	unExecutedEventsFile = "unexecuted_events.json"
)

func saveUnExecutedEvents(events []Event) error {
	data, err := json.Marshal(events)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(unExecutedEventsFile, data, 0644)
}

func loadUnExecutedEvents() []Event {
	data, err := ioutil.ReadFile(unExecutedEventsFile)
	if err != nil {
		return nil
	}
	var events []Event
	err = json.Unmarshal(data, &events)
	if err != nil {
		return nil
	}
	return events
}

func GetInstance() *Manager {
	once.Do(func() {
		f := make(map[EventType][]FuncCallback)
		evt := make(chan Event, 100)
		lock := sync.Mutex{}
		wg := sync.WaitGroup{}
		limiter := NewConnLimiter(ConcurrentNum)
		sendEventMutex := sync.Mutex{}
		unExecutedL := sync.Mutex{}                                                                                                                         // initialize new mutex
		instance = &Manager{functions: f, eventChain: evt, L: &lock, Limiter: limiter, wg: &wg, unExecutedL: &unExecutedL, sendEventMutex: &sendEventMutex} // add unExecutedL field
	})
	return instance
}

type FuncCallback func(event Event)

func (em *Manager) AddListener(eventType EventType, f FuncCallback) {
	em.functions[eventType] = append(em.functions[eventType], f)
}

func (em *Manager) ClearListener() {
	em.L.Lock()
	defer em.L.Unlock()
	for k := range em.functions {
		delete(em.functions, k)
	}
}

func (em *Manager) SendEvent(event Event) {
	em.sendEventMutex.Lock()
	event.ID = uuid.New().String()
	em.sendEventMutex.Unlock()
	em.eventChain <- event
}

func (em *Manager) Stop() {
	done <- true
	em.ClearListener()
	close(em.eventChain)
	err := saveUnExecutedEvents(em.unExecuted)
	if err != nil {
	}
	em.unExecuted = nil
}

func (em *Manager) EventWorker(ctx context.Context, token chan struct{}) {
	defer func() {
		if r := recover(); r != nil {
			panic(r)
		}
	}()
	for event := range em.eventChain {
		select {
		case <-ctx.Done():
			return
		case token <- struct{}{}:
			em.Limiter.GetConn()
			em.Limiter.GetConn()
			localEvent := event
			// ...
			em.Limiter.ReleaseConn()
			<-token
			if _, ok := em.functions[localEvent.Type]; ok {
				for _, fn := range em.functions[localEvent.Type] {
					fn(localEvent)
				}
				em.unExecutedL.Lock()
				for i, e := range em.unExecuted {
					if e.ID == localEvent.ID {
						em.unExecuted = append(em.unExecuted[:i], em.unExecuted[i+1:]...)
						break
					}
				}
				em.unExecutedL.Unlock()
			} else {
				em.unExecutedL.Lock()
				em.unExecuted = append(em.unExecuted, localEvent)
				em.unExecutedL.Unlock()
			}
			em.Limiter.ReleaseConn()
			<-token
		}
	}
}

func (em *Manager) Start() {
	done = make(chan bool, 1)
	em.unExecuted = []Event{}
	go em.__Start()
}

func (em *Manager) __Start() {
	//defer em.L.Unlock()
	//em.L.Lock() // 只允许一个Worker运行
	ctx, cancel := context.WithCancel(context.Background())
	token := make(chan struct{}, ConcurrentNum)
	for i := 0; i < ConcurrentNum; i++ {
		go em.EventWorker(ctx, token) // 死循环
	}
	em.unExecuted = loadUnExecutedEvents()
	for _, event := range em.unExecuted {
		em.eventChain <- event
	}
	<-done
	cancel()
}
