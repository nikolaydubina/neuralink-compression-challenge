package main

import "sort"

type CacheConfig struct {
	Size int
}

type cacheEntry struct {
	key   uint16
	count int
}

// Cache is not as efficient, but is ok for prototype.
type Cache struct {
	config CacheConfig
	order  []cacheEntry
}

func NewCache(config CacheConfig) *Cache {
	return &Cache{
		config: config,
		order:  make([]cacheEntry, 0, config.Size),
	}
}

func (s *Cache) Pop() {
	if len(s.order) == 0 {
		return
	}
	s.order = s.order[:len(s.order)-1]
}

func (s *Cache) Add(v uint16) {
	if i := s.Index(v); i >= 0 {
		s.order[i].count++
	} else {
		if s.IsFull() {
			s.Pop()
		}
		s.order = append(s.order, cacheEntry{key: v, count: 1})
	}
	sort.SliceStable(s.order, func(i, j int) bool { return s.order[i].count > s.order[j].count })
}

func (s *Cache) Index(v uint16) int {
	for i, q := range s.order {
		if q.key == v {
			return i
		}
	}
	return -1
}

func (s *Cache) At(i int) uint16 { return s.order[i].key }

func (s *Cache) IsFull() bool { return len(s.order) >= s.config.Size }
