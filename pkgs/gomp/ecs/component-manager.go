/*
This Source Code Form is subject to the terms of the Mozilla
Public License, v. 2.0. If a copy of the MPL was not distributed
with this file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

package ecs

import (
	"github.com/alphadose/haxmap"
	"github.com/negrel/assert"
)

const preallocatedCapacity = 1 << 14

type ComponentManager[T any] struct {
	data          []T
	entities      []EntityID
	lookup        *haxmap.Map[EntityID, int]
	worldMask     *ComponentManager[ComponentBitArray256]
	isInitialized bool
	ID            ComponentID
}

func CreateComponentManager[T any](id ComponentID) *ComponentManager[T] {
	return &ComponentManager[T]{
		data:          make([]T, 0, preallocatedCapacity),
		entities:      make([]EntityID, 0, preallocatedCapacity),
		lookup:        haxmap.New[EntityID, int](preallocatedCapacity),
		isInitialized: true,
		ID:            id,
	}
}

func (c *ComponentManager[T]) registerComponentMask(mask *ComponentManager[ComponentBitArray256]) {
	c.worldMask = mask
}

func (c *ComponentManager[T]) getId() ComponentID {
	return c.ID
}

func (c *ComponentManager[T]) Create(entity EntityID, value T) *T {
	// ComponentManager must be initialized with CreateComponentManager()
	assert.True(c.isInitialized)

	// INVALID ENTITY!
	assert.True(entity != -1)

	// Only one of component per enity allowed!
	assert.False(c.lookUpHas(entity))

	// Entity Count must always be the same as the number of components!
	assert.True(len(c.entities) == len(c.data))
	assert.True(len(c.data) == int(c.lookup.Len()))

	var index = len(c.data)

	c.lookup.Set(entity, index)

	c.data = append(c.data, value)
	c.entities = append(c.entities, entity)

	if c.ID != ENTITY_COMPONENT_MASK_ID {
		mask := c.worldMask.Get(entity)
		mask.Set(c.ID)
	}

	return &c.data[index]
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

	return &c.data[index]
}

func (c *ComponentManager[T]) Remove(entity EntityID) {
	// ComponentManager must be initialized with CreateComponentManager()
	assert.True(c.isInitialized)

	// INVALID ENTITY!
	assert.False(entity == -1)

	// ENTITY HAS NO COMPONENT!
	assert.True(c.lookUpHas(entity))

	index, _ := c.lookup.Get(entity)

	lastIndex := len(c.data) - 1
	if index < lastIndex {
		// Swap the the dead element with the last one
		c.data[index] = c.data[lastIndex]
		c.entities[index] = c.entities[lastIndex]

		// Update the lookup table
		c.lookup.Set(c.entities[index], index)
	}

	// Shrink the container
	newSize := len(c.data) - 1
	c.data = c.data[:newSize]
	c.entities = c.entities[:newSize]
	c.lookup.Del(entity)

	// Entity Count must always be the same as the number of components!
	assert.True(len(c.entities) == len(c.data))
	assert.True(len(c.data) == int(c.lookup.Len()))
}

func (c *ComponentManager[T]) All(yield func(EntityID, *T) bool) {
	// ComponentManager must be initialized with CreateComponentManager()
	assert.True(c.isInitialized)

	// Entity Count must always be the same as the number of components!
	assert.True(len(c.entities) == len(c.data))
	assert.True(len(c.data) == int(c.lookup.Len()))

	for index := len(c.data) - 1; index >= 0; index-- {
		id := c.entities[index]
		value := &c.data[index]
		if !yield(id, value) {
			break
		}
	}
}

func (c *ComponentManager[T]) AllParallel(yield func(EntityID, *T) bool) {
	// ComponentManager must be initialized with CreateComponentManager()
	assert.True(c.isInitialized)

	// Entity Count must always be the same as the number of components!
	assert.True(len(c.entities) == len(c.data))
	assert.True(len(c.data) == int(c.lookup.Len()))

	for index := len(c.data) - 1; index >= 0; index-- {
		id := c.entities[index]
		value := &c.data[index]
		if !yield(id, value) {
			break
		}
	}
}

func (c *ComponentManager[T]) AllData(yield func(*T) bool) {
	// ComponentManager must be initialized with CreateComponentManager()
	assert.True(c.isInitialized)

	// Entity Count must always be the same as the number of components!
	assert.True(len(c.entities) == len(c.data))
	assert.True(len(c.data) == int(c.lookup.Len()))

	for index := len(c.data) - 1; index >= 0; index-- {
		value := &c.data[index]
		if !yield(value) {
			break
		}
	}
}

func (c *ComponentManager[T]) AllDataParallel(yield func(*T) bool) {
	// ComponentManager must be initialized with CreateComponentManager()
	assert.True(c.isInitialized)

	// Entity Count must always be the same as the number of components!
	assert.True(len(c.entities) == len(c.data))
	assert.True(len(c.data) == int(c.lookup.Len()))

	for index := len(c.data) - 1; index >= 0; index-- {
		value := &c.data[index]
		if !yield(value) {
			break
		}
	}
}

func (c *ComponentManager[T]) Len() int {
	// ComponentManager must be initialized with CreateComponentManager()
	assert.True(c.isInitialized)

	return len(c.data)
}

func (c *ComponentManager[T]) Clean() {
	// TODO
}

func (c ComponentManager[T]) lookUpHas(key EntityID) bool {
	// ComponentManager must be initialized with CreateComponentManager()
	assert.True(c.isInitialized)

	_, ok := c.lookup.Get(key)
	return ok
}
