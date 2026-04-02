package common

import "sync"

type SimpleMap[K comparable, T any] struct {
	mutex sync.RWMutex
	fn    func(*T) K
	list  map[K]*T
}

func NewSimpleMap[K comparable, T any](f func(*T) K) SimpleMap[K, T] {
	return SimpleMap[K, T]{
		list: make(map[K]*T),
		fn:   f,
	}
}

func (m *SimpleMap[K, T]) Add(v *T) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.list[m.fn(v)] = v
}

func (m *SimpleMap[K, T]) Find(k K) (v *T) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	v, _ = m.list[k]
	return
}

func (m *SimpleMap[K, T]) Remove(k K) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.list, k)
}

func (m *SimpleMap[K, T]) Do(fn func(item *T) bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	for _, v := range m.list {
		if !fn(v) {
			break
		}
	}
	return
}

func (m *SimpleMap[K, T]) ListAll() (l []*T) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	for _, v := range m.list {
		l = append(l, v)
	}
	return
}

func (m *SimpleMap[K, T]) Size() int {
	return len(m.list)
}

type MultiMap[T any] struct {
	mutex sync.RWMutex
	maps  map[uint8]map[string]*T
	fn    func(*T, uint8) string
}

func NewMultiMap[T any](idtypes []uint8, fn func(*T, uint8) string) (ret MultiMap[T]) {
	ret.maps = make(map[uint8]map[string]*T)
	ret.fn = fn
	for _, t := range idtypes {
		ret.maps[t] = make(map[string]*T)
	}
	return
}

// update an id for an existing item
func (m *MultiMap[T]) UpdateId(t uint8, i *T) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if id := m.fn(i, t); len(id) > 0 {
		if onemap, ok := m.maps[t]; ok {
			onemap[id] = i
			return true
		}
	}
	return false
}

// add  or update an item
func (m *MultiMap[T]) Update(i *T) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	for t, onemap := range m.maps {
		if id := m.fn(i, t); len(id) > 0 {
			onemap[id] = i
		}
	}
}

func (m *MultiMap[T]) Find(id string, idtype uint8) (i *T) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if onemap, ok := m.maps[idtype]; ok {
		i, _ = onemap[id]
	}
	return
}

func (m *MultiMap[T]) Remove(i *T) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	for t, onemap := range m.maps {
		if id := m.fn(i, t); len(id) > 0 {
			delete(onemap, id)
		}
	}
}

func (m *MultiMap[T]) Do(t uint8, fn func(*T) bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if onemap, ok := m.maps[t]; ok {
		for _, i := range onemap {
			if !fn(i) {
				break
			}
		}
	}
	return
}

func (m *MultiMap[T]) ListAll(t uint8) (l []*T) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if onemap, ok := m.maps[t]; ok {
		for _, i := range onemap {
			l = append(l, i)
		}
	}
	return
}

func (m *MultiMap[T]) Size(t uint8) int {
	if onemap, ok := m.maps[t]; ok {
		return len(onemap)
	}
	return 0
}
