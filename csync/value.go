package csync

import (
	"sync/atomic"
)

//////////////////////////////////////////////////

type Value[T any] struct {
	av atomic.Value
}

func (v *Value[T]) Load() (value T) {
	if v == nil {
		return
	}

	val, ok := v.av.Load(), false
	if value, ok = val.(T); ok {
		return
	}

	return
}

func (v *Value[T]) Store(value T) {
	if v == nil {
		return
	}

	v.av.Store(value)
}

func (v *Value[T]) Swap(new T) (old T) {
	if v == nil {
		return
	}

	oldVal, ok := v.av.Swap(new), false
	if old, ok = oldVal.(T); ok {
		return
	}

	return
}

func (v *Value[T]) CompareAndSwap(old T, new T) (swapped bool) {
	if v == nil {
		return
	}

	swapped = v.av.CompareAndSwap(old, new)

	return
}
