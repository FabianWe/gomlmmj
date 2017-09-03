// The MIT License (MIT)

// Copyright (c) 2017 Fabian Wenzelmann

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package gomlmmj

import "sync"

type ListManager struct {
	mutex sync.RWMutex
	lists map[string]*sync.RWMutex
}

func NewListManager() *ListManager {
	return &ListManager{}
}

func (lm *ListManager) AddList(name string) bool {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()
	if _, hasList := lm.lists[name]; hasList {
		return false
	}
	lm.lists[name] = new(sync.RWMutex)
	return true
}

func (lm *ListManager) RemoveList(name string) bool {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()
	if _, hasList := lm.lists[name]; !hasList {
		return false
	}
	delete(lm.lists, name)
	return true
}

func (lm *ListManager) ReadList(name string) (bool, func()) {
	lm.mutex.RLock()
	defer lm.mutex.RUnlock()
	m, hasList := lm.lists[name]
	if !hasList {
		f := func() {}
		return false, f
	}
	// lock the mutex and return a function that unlocks it
	m.RLock()
	f := func() {
		m.RUnlock()
	}
	return true, f
}

func (lm *ListManager) WriteList(name string) (bool, func()) {
	lm.mutex.RLock()
	defer lm.mutex.RUnlock()
	m, hasList := lm.lists[name]
	if !hasList {
		f := func() {}
		return false, f
	}
	// lock the mutex and return a function that unlocks it
	m.Lock()
	f := func() {
		m.Unlock()
	}
	return true, f
}

func (lm *ListManager) Init(spool string) error {
	// remove old mutexes in any case
	lm.lists = make(map[string]*sync.RWMutex)
	lists, err := GetLists(spool)
	if err != nil {
		return err
	}
	for _, list := range lists {
		lm.AddList(list)
	}
	return nil
}
