/*
This Source Code Form is subject to the terms of the Mozilla
Public License, v. 2.0. If a copy of the MPL was not distributed
with this file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

package ecs

import (
	"fmt"
	"math/big"
	"sync"
)

type AnyComponentTypePtr[W any] interface {
	register(*W, ComponentID) AnyComponentInstancesPtr
}

type AnyComponentInstancesPtr interface {
	registerComponentMask(mask *ComponentManager[big.Int])
	getId() ComponentID
	Remove(EntityID)
	Clean()
	Has(EntityID) bool
}

type ComponentType[T any] struct {
	worldComponents map[*World]WorldComponents[T]

	wg *sync.WaitGroup
	mx *sync.Mutex
}

func CreateComponent[T any]() ComponentType[T] {
	component := ComponentType[T]{}
	component.worldComponents = make(map[*World]WorldComponents[T])
	component.wg = new(sync.WaitGroup)
	component.mx = new(sync.Mutex)

	return component
}

func (c *ComponentType[T]) Instances(ecs *World) WorldComponents[T] {
	if value, ok := c.worldComponents[ecs]; ok {
		return value
	}

	panic(fmt.Sprintf("Component <%T> is not registered in <%s> world", c, ecs.Title))
}

func (c *ComponentType[T]) register(ecs *World, id ComponentID) AnyComponentInstancesPtr {
	newInstances := NewSparseSet[T, EntityID]()

	newComponents := WorldComponents[T]{
		mx:            new(sync.Mutex),
		ID:            id,
		maskComponent: ecs.entityComponentMask,
		instances:     &newInstances,
	}

	c.worldComponents[ecs] = newComponents

	return &newComponents
}

type WorldComponents[T any] struct {
	mx            *sync.Mutex
	ID            ComponentID
	maskComponent *SparseSet[ComponentBitArray256, EntityID]
	instances     *SparseSet[T, EntityID]
}

func (c *WorldComponents[T]) getId() ComponentID {
	return c.ID
}

func (c *WorldComponents[T]) registerComponentMask(mask *ComponentManager[big.Int]) {
}

func (c *WorldComponents[T]) Get(entity EntityID) (T, bool) {
	instance, ok := c.instances.Get(entity)

	return instance, ok
}

func (c *WorldComponents[T]) Has(entity EntityID) bool {
	_, ok := c.instances.Get(entity)
	return ok
}

func (c *WorldComponents[T]) GetPtr(entity EntityID) (value *T) {
	value = c.instances.GetPtr(entity)
	return value
}

func (c *WorldComponents[T]) Set(entityID EntityID, data T) *T {
	c.mx.Lock()
	defer c.mx.Unlock()

	var newinstance = c.instances.Set(entityID, data)

	mask := c.maskComponent.GetPtr(entityID)
	mask.Set(c.ID)

	return newinstance
}

func (c *WorldComponents[T]) Remove(entityID EntityID) {
	c.instances.SoftDelete(entityID)
	mask := c.maskComponent.GetPtr(entityID)
	mask.Unset(c.ID)
}

func (c *WorldComponents[T]) Clean() {
	c.instances.Clean()
	c.maskComponent.Clean()
}

func (c *WorldComponents[T]) All(yield func(EntityID, *T) bool) {
	c.instances.All(yield)
}

func (c *WorldComponents[T]) AllParallel(yield func(EntityID, *T) bool) {
	c.instances.AllParallel(yield)
}

func (c *WorldComponents[T]) AllData(yield func(*T) bool) {
	c.instances.AllData(yield)
}

func (c *WorldComponents[T]) AllDataParallel(yield func(*T) bool) {
	c.instances.AllDataParallel(yield)
}
func (c *WorldComponents[T]) Len() int {
	return c.instances.Len()
}

// To use more threads we need to prespawn goroutines for each component
// var threads = runtime.NumCPU() * 2

// TODO EachParallel()
// func (c *Component[T]) Each(ecs *ECS, callback func(*Entity, T)) {
// 	ecsInstances, _ := c.instances.Get(int(ecs.ID))

// 	for _, instance := range ecsInstances.Iter() {
// 		callback(instance.Entity, instance.Data)
// 	}

// 	// for _, s := range c.Instances {
// 	// 	s.Clean()
// 	// }
// }

// func (c *Component[T]) EachParallel(ecs *ECS, callback func(*Entity, *T)) {
// 	arr := c.Instances[ecs.ID].denseData.buckets
// 	for _, b := range arr {
// 		c.parallelCallback(callback, b.data)
// 	}

// 	for _, s := range c.Instances {
// 		s.Clean()
// 	}
// }

// func (c *Component[T]) parallelCallback(callback func(*Entity, *T), data []ComponentInstance[T]) {
// 	for j := 0; j < len(data); j++ {
// 		callback(data[j].Entity, &data[j].Data)
// 	}
// }
