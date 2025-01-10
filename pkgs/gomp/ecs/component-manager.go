/*
This Source Code Form is subject to the terms of the Mozilla
Public License, v. 2.0. If a copy of the MPL was not distributed
with this file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

package ecs

import (
	"iter"
	"math/big"
	"sync"

	"github.com/negrel/assert"
)

type ComponentManager[T any] struct {
	mx            *sync.Mutex
	components    *PagedArray[T]
	entities      *PagedArray[EntityID]
	lookup        *PagedMap[EntityID, int32]
	ID            ComponentID
	isInitialized bool
}

func CreateComponentManager[T any](id ComponentID) *ComponentManager[T] {
	inst := &ComponentManager[T]{}
	inst.Init()
	return inst
}

func (c *ComponentManager[T]) Init() {
	// (aled93): Init need for reflection because calling generic function
	// using reflection isn't possible as I know
	if c.isInitialized {
		return
	}

	c.components = NewPagedArray[T]()
	c.entities = NewPagedArray[EntityID]()
	c.lookup = NewPagedMap[EntityID, int32]()
	c.isInitialized = true
	c.mx = new(sync.Mutex)
}

func (c *ComponentManager[T]) registerComponentMask(mask *ComponentManager[big.Int]) {
	// c.worldMask = mask
}

func (c *ComponentManager[T]) getId() ComponentID {
	return c.ID
}

func (c *ComponentManager[T]) Create(entity EntityID, value T) (returnValue *T) {
	c.mx.Lock()
	defer c.mx.Unlock()

	// ComponentManager must be initialized with CreateComponentManager()
	assert.True(c.isInitialized)

	// INVALID ENTITY!
	assert.True(entity != -1)

	// Only one of component per enity allowed!
	assert.False(c.Has(entity))

	// Entity Count must always be the same as the number of components!
	assert.True(c.entities.Len() == c.components.Len())
	assert.True(c.components.Len() == c.lookup.Len())

	var index = c.components.Len()

	c.lookup.Set(entity, index)
	c.entities.Append(entity)
	return c.components.Append(value)
}

func (c *ComponentManager[T]) Get(entity EntityID) *T {
	// ComponentManager must be initialized with CreateComponentManager()
	assert.True(c.isInitialized)

	// INVALID ENTITY!
	assert.False(entity == -1)

	index, ok := c.lookup.Get(entity)
	if !ok {
		return nil
	}

	return c.components.Get(index)
}

func (c *ComponentManager[T]) Remove(entity EntityID) {
	c.mx.Lock()
	defer c.mx.Unlock()

	// ComponentManager must be initialized with CreateComponentManager()
	assert.True(c.isInitialized)

	// INVALID ENTITY!
	assert.False(entity == -1)

	// ENTITY HAS NO COMPONENT!
	assert.True(c.Has(entity))

	index, _ := c.lookup.Get(entity)

	lastIndex := c.components.Len() - 1
	if index < lastIndex {
		// Swap the the dead element with the last one
		c.components.Swap(index, lastIndex)
		newSwappedEntityId, _ := c.entities.Swap(index, lastIndex)
		assert.True(newSwappedEntityId != nil)

		// Update the lookup table
		c.lookup.Set(*newSwappedEntityId, index)
	}

	// Shrink the container
	c.components.SoftReduce()
	c.entities.SoftReduce()

	c.lookup.Delete(entity)

	// Entity Count must always be the same as the number of components!
	assert.True(c.entities.Len() == c.components.Len())
	assert.True(c.components.Len() == c.lookup.Len())
}

func (c *ComponentManager[T]) Has(entity EntityID) bool {
	_, ok := c.lookup.Get(entity)
	return ok
}

func (c *ComponentManager[T]) All(yield func(EntityID, *T) bool) {
	// ComponentManager must be initialized with CreateComponentManager()
	assert.True(c.isInitialized)

	// Entity Count must always be the same as the number of components!
	assert.True(c.entities.Len() == c.components.Len())
	assert.True(c.components.Len() == c.lookup.Len())

	nextData, stopData := iter.Pull(c.components.AllData)
	defer stopData()

	nextEntity, stopEntity := iter.Pull(c.entities.AllData)
	defer stopEntity()

	for {
		data, ok := nextData()
		if !ok {
			break
		}
		entityId, ok := nextEntity()
		if !ok {
			break
		}
		assert.True(entityId != nil)
		entId := *entityId
		shouldContinue := yield(entId, data)
		if !shouldContinue {
			break
		}
	}
}

func (c *ComponentManager[T]) AllParallel(yield func(EntityID, *T) bool) {
	// ComponentManager must be initialized with CreateComponentManager()
	assert.True(c.isInitialized)

	// Entity Count must always be the same as the number of components!
	assert.True(c.entities.Len() == c.components.Len())
	assert.True(c.components.Len() == c.lookup.Len())

	c.components.AllParallel(func(i int32, t *T) bool {
		entId := c.entities.Get(i)
		assert.True(entId != nil)
		return yield(*entId, t)
	})
}

func (c *ComponentManager[T]) AllData(yield func(*T) bool) {
	// ComponentManager must be initialized with CreateComponentManager()
	assert.True(c.isInitialized)

	// Entity Count must always be the same as the number of components!
	assert.True(c.entities.Len() == c.components.Len())
	assert.True(c.components.Len() == c.lookup.Len())

	c.components.AllData(yield)
}

func (c *ComponentManager[T]) AllDataParallel(yield func(*T) bool) {
	// ComponentManager must be initialized with CreateComponentManager()
	assert.True(c.isInitialized)

	// Entity Count must always be the same as the number of components!
	assert.True(c.entities.Len() == c.components.Len())
	assert.True(c.components.Len() == c.lookup.Len())

	c.components.AllDataParallel(yield)
}

func (c *ComponentManager[T]) Len() int32 {
	// ComponentManager must be initialized with CreateComponentManager()
	assert.True(c.isInitialized)

	return c.components.Len()
}

func (c *ComponentManager[T]) Clean() {
	// c.components.Clean()
	// c.entities.Clean()
}
