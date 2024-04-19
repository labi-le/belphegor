// https://github.com/indigo-web/utils/blob/master/pool/objectpool.go

package pool

import "sync"

// ObjectPool is a generic analogue of sync.Pool, except it does not provide
// thread-safety. Concurrent access may lead to UB
type ObjectPool[T any] struct {
	queue []T
	New   func() T
	mutex sync.Mutex // Добавляем мьютекс для синхронизации доступа
}

func NewObjectPool[T any](queueSize int) *ObjectPool[T] {
	return &ObjectPool[T]{
		queue: make([]T, 0, queueSize),
	}
}

func (o *ObjectPool[T]) Acquire() (obj T) {
	o.mutex.Lock()         // Блокируем доступ к очереди
	defer o.mutex.Unlock() // Обязательно разблокируем после возврата объекта

	if len(o.queue) != 0 {
		obj = o.queue[len(o.queue)-1]
		o.queue = o.queue[:len(o.queue)-1]
	}

	if o.New != nil {
		return o.New()
	}

	return obj
}

func (o *ObjectPool[T]) Release(obj T) {
	o.mutex.Lock()         // Блокируем доступ к очереди
	defer o.mutex.Unlock() // Обязательно разблокируем после освобождения объекта

	if len(o.queue) == cap(o.queue) {
		return
	}

	o.queue = append(o.queue, obj)
}
