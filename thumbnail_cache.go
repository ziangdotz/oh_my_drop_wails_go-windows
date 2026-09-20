package main

import (
	"container/list"
	"sync"
)

// thumbnailCacheCapacity 限制缩略图缓存的最大条目数，防止长时间使用导致内存无限增长。
const thumbnailCacheCapacity = 256

// thumbnailLRU 是一个并发安全、带容量上限的 LRU 缓存，用于缓存文件缩略图。
type thumbnailLRU struct {
	mu       sync.Mutex
	capacity int
	ll       *list.List
	items    map[string]*list.Element
}

type thumbnailEntry struct {
	key string
	val Thumbnail
}

func newThumbnailLRU(capacity int) *thumbnailLRU {
	if capacity <= 0 {
		capacity = thumbnailCacheCapacity
	}
	return &thumbnailLRU{
		capacity: capacity,
		ll:       list.New(),
		items:    make(map[string]*list.Element, capacity),
	}
}

// Load 返回缓存中的缩略图；命中时将条目提升为最近使用。
func (c *thumbnailLRU) Load(key string) (Thumbnail, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[key]
	if !ok {
		return Thumbnail{}, false
	}
	c.ll.MoveToFront(el)
	return el.Value.(*thumbnailEntry).val, true
}

// Store 写入缩略图；若超出容量则淘汰最久未使用的条目。
func (c *thumbnailLRU) Store(key string, val Thumbnail) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		el.Value.(*thumbnailEntry).val = val
		c.ll.MoveToFront(el)
		return
	}
	el := c.ll.PushFront(&thumbnailEntry{key: key, val: val})
	c.items[key] = el
	if c.ll.Len() > c.capacity {
		c.removeOldest()
	}
}

// Clear 清空全部缓存。
func (c *thumbnailLRU) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ll.Init()
	c.items = make(map[string]*list.Element, c.capacity)
}

// removeOldest 淘汰链表尾部（最久未使用）的条目。调用方需持有锁。
func (c *thumbnailLRU) removeOldest() {
	el := c.ll.Back()
	if el == nil {
		return
	}
	c.ll.Remove(el)
	delete(c.items, el.Value.(*thumbnailEntry).key)
}
