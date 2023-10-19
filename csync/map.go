package csync

import "sync"

//////////////////////////////////////////////////

type Map[K comparable, V any] struct {
	m sync.Map
}

func (mm *Map[K, V]) Has(key K) bool {
	if mm == nil {
		return false
	}

	if _, ok := mm.m.Load(key); ok {
		return true
	}

	return false
}

func (mm *Map[K, V]) CompareAndDelete(key K, old V) (deleted bool) {
	if mm == nil {
		return
	}

	return mm.m.CompareAndDelete(key, old)
}

func (mm *Map[K, V]) CompareAndSwap(key K, old V, new V) bool {
	if mm == nil {
		return false
	}

	return mm.m.CompareAndSwap(key, old, new)
}

func (mm *Map[K, V]) Delete(key K) {
	if mm == nil {
		return
	}

	mm.m.Delete(key)
}

func (mm *Map[K, V]) Load(key K) (value V, ok bool) {
	if mm == nil {
		return
	}

	var v any

	v, ok = mm.m.Load(key)
	if !ok {
		return
	}

	value, ok = v.(V)
	return
}

func (mm *Map[K, V]) LoadAndDelete(key K) (value V, loaded bool) {
	if mm == nil {
		return
	}

	var v any

	v, loaded = mm.m.LoadAndDelete(key)
	if !loaded {
		return
	}

	value, ok := v.(V)
	if ok {
		return value, loaded
	}

	return
}

func (mm *Map[K, V]) LoadOrStore(key K, value V) (actual V, loaded bool) {
	if mm == nil {
		return
	}

	var v any

	v, loaded = mm.m.LoadOrStore(key, value)
	if !loaded {
		return value, false
	}

	actual, loaded = v.(V)
	return actual, true
}

func (mm *Map[K, V]) Range(f func(key K, value V) bool) {
	if mm == nil {
		return
	}

	mm.m.Range(func(k any, v any) bool {
		key, ok := k.(K)
		if !ok {
			return false
		}

		value, ok := v.(V)
		if !ok {
			return false
		}

		return f(key, value)
	})
}

func (mm *Map[K, V]) Store(key K, value V) {
	if mm == nil {
		return
	}

	mm.m.Store(key, value)
}

func (mm *Map[K, V]) Swap(key K, value V) (previous V, loaded bool) {
	if mm == nil {
		return
	}

	var v any

	v, loaded = mm.m.Swap(key, value)
	if !loaded {
		return
	}

	previous, ok := v.(V)
	if !ok {
		return
	}

	return
}
