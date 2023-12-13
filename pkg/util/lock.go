package util

import "sync"

// MultiLocker -
type MultiLocker struct {
	M   map[string]*sync.Mutex
	Smu sync.Mutex
}

// Lock -
func (l *MultiLocker) Lock(name string) {
	l.Smu.Lock()
	mutex, ok := l.M[name]
	if !ok {
		mutex = new(sync.Mutex)
		l.M[name] = mutex
	}
	l.Smu.Unlock()
	mutex.Lock()
}

// Unlock -
func (l *MultiLocker) Unlock(name string) {
	l.Smu.Lock()
	defer l.Smu.Unlock()
	l.M[name].Unlock()
}
