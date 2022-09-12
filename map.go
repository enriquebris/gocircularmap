package gocircularmap

import (
	"errors"
	"fmt"
	"sync"
)

type Map struct {
	// map to store entries
	mp map[any]any
	// circular channel
	entriesChan chan any
	// mutex
	mutex sync.RWMutex
	// total entries
	totalEntries uint
	// max allowed entries
	maxEntries uint
}

// NewMap returns a new instance of *Map
func NewMap(maxEntries uint) *Map {
	mp := &Map{}
	mp.initialize(maxEntries)

	return mp
}

func (st *Map) initialize(maxEntries uint) {
	st.mp = make(map[any]any)
	st.maxEntries = maxEntries
	st.entriesChan = make(chan any, st.maxEntries)
}

// Store sets the value for a key.
func (st *Map) Store(key, value any) error {
	st.mutex.Lock()
	defer st.mutex.Unlock()

	if st.totalEntries >= st.maxEntries {
		// get key to remove
		keyToRemove, err := st.getFirstKey()
		if err != nil {
			return err
		}

		// remove entry && decrement total counter
		st.deleteNoLock(keyToRemove)
	}

	// add entry
	st.mp[key] = value

	// increment total
	st.totalEntries++

	// enqueue entry in circular queue
	select {
	case st.entriesChan <- key:
		// everything is fine
	default:
		// rollback: remove key-value
		st.deleteNoLock(key)

		return fmt.Errorf("error adding entry: %v", key)
	}

	return nil
}

// Load returns the value stored in the map for a key, or nil if no value is present. The ok result indicates whether value was found in the map.
func (st *Map) Load(key any) (value any, ok bool) {
	st.mutex.RLock()
	defer st.mutex.RUnlock()

	if value, ok := st.mp[key]; ok {
		return value, ok
	}

	return nil, false
}

// Delete deletes the value for a key.
func (st *Map) Delete(key any) {
	st.mutex.Lock()
	defer st.mutex.Unlock()

	st.deleteNoLock(key)
}

// deleteNoLock deletes an entry given the key. If there is no such element, deleteNoLock is a no-op.
// This method does not lock the instance.
func (st *Map) deleteNoLock(key any) {
	if _, ok := st.mp[key]; ok {
		// remove entry
		delete(st.mp, key)

		// decrement total
		st.totalEntries--
	}
}

// getFirstKey returns the first entry's key from current entries, if any
func (st *Map) getFirstKey() (any, error) {
	select {
	case key := <-st.entriesChan:
		return key, nil
	default:
		return nil, errors.New("no available first entry key")
	}
}
