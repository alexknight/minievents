package minevents

import (
	"sync"
)

var (
	limiter *ConnLimiter
)

type ConnLimiter struct {
	count int
	limit int
	mutex sync.Mutex
}

func NewConnLimiter(limit int) *ConnLimiter {
	return &ConnLimiter{
		limit: limit,
	}
}

func (l *ConnLimiter) GetConn() bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.count >= l.limit {
		return false
	}

	l.count++
	return true
}

func (l *ConnLimiter) ReleaseConn() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.count--
}
