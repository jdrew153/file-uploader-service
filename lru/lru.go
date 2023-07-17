package lru

import (
	"container/list"
	"fmt"
	"sync"
)

type LRUCache struct {
	Cache    map[string]*list.Element
	LinkList *list.List
	// File number capacity
	Capacity int
	// File size capacity
	CurrCacheSize int
	Lock          sync.Mutex
}

func Constructor(capacity int) LRUCache {
	return LRUCache{
		Cache:         make(map[string]*list.Element, capacity),
		LinkList:      list.New(),
		Capacity:      capacity,
		CurrCacheSize: 0,
	}
}

func (this *LRUCache) Get(key string) int {
	this.Lock.Lock()
	defer this.Lock.Unlock()

	elem, ok := this.Cache[key]

	if !ok {
		return -1
	} else {
		this.LinkList.MoveToFront(elem)
		value, ok := elem.Value.([]int)
		if !ok || len(value) < 2 {
			return -1
		}

		return value[1]
	}
}

func (this *LRUCache) Put(key string, value int) int {

	this.Lock.Lock()
	defer this.Lock.Unlock()

	val, ok := this.Cache[key]
	fmt.Println("Cache size: ", this.CurrCacheSize >=  100000000)
	if this.CurrCacheSize >= 100000000 {
		fmt.Println("Cache is full")
		this.RemoveHeaviest()
		return -1
	}

	if ok {
								
		if this.LinkList.Len() == this.Capacity {
			fmt.Println("Cache has reached len capacity")
			lastElem := this.LinkList.Back()
			this.LinkList.Remove(lastElem)
			delete(this.Cache, lastElem.Value.([]string)[0])
			fmt.Println("Removed: ", lastElem.Value.([]string)[0])
			this.CurrCacheSize -= lastElem.Value.([]int)[1]
			return lastElem.Value.([]int)[0]
		}

		fmt.Println("Updating existing elem hit, moving to front.. :", val)
		elem := this.Cache[key]
		elem.Value = map[string]int{key: value}
		this.LinkList.MoveToFront(elem)
		this.CurrCacheSize += elem.Value.([]int)[1]

	} else {

		fmt.Println("New elem being added to cache..")
		this.LinkList.PushFront(map[string]int{key: value})
		this.CurrCacheSize += value
		this.Cache[key] = this.LinkList.Front()
	}

	return -1
}
// base 64 data:video/mp4;base64

func (this *LRUCache) RemoveHeaviest() {

	this.Lock.Lock()
	defer this.Lock.Unlock()

	heaviest := 0
	heaviestKey := ""
	for k, v := range this.Cache {
		value, ok := v.Value.([]int)
		if !ok || len(value) < 2 {
			continue // Skip if the value is not of type []int or doesn't have sufficient elements
		}

		if value[1] > heaviest {
			heaviest = value[1]
			heaviestKey = k
		}
	}

	if heaviestKey != "" {
		elem, ok := this.Cache[heaviestKey]
		if ok {
			this.LinkList.Remove(elem)
			delete(this.Cache, heaviestKey)
			this.CurrCacheSize -= heaviest
		}
	}
}
