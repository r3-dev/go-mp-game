/*
This Source Code Form is subject to the terms of the Mozilla
Public License, v. 2.0. If a copy of the MPL was not distributed
with this file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

package ecs

import (
	"sync"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/negrel/assert"
)

const ENTITY_COMPONENT_MASK_ID ComponentID = 1<<8 - 1

type GenericWorld[T any, S any] struct {
	Components *T
	Systems    *S

	components    []AnyComponentInstancesPtr
	updateSystems [][]AnyUpdateSystem[GenericWorld[T, S]]
	drawSystems   [][]AnyDrawSystem[GenericWorld[T, S]]

	entityComponentMask *ComponentManager[ComponentBitArray256]
	deletedEntityIDs    []EntityID
	lastEntityID        EntityID

	tick int
	ID   ECSID

	wg   *sync.WaitGroup
	size int32
}

func CreateGenericWorld[T any, S any](id ECSID, components *T, systems *S) GenericWorld[T, S] {
	maskSet := CreateComponentManager[ComponentBitArray256](ENTITY_COMPONENT_MASK_ID)
	ecs := GenericWorld[T, S]{
		ID:                  id,
		Components:          components,
		Systems:             systems,
		wg:                  new(sync.WaitGroup),
		deletedEntityIDs:    make([]EntityID, 0, PREALLOC_DELETED_ENTITIES),
		entityComponentMask: maskSet,
	}

	return ecs
}

func (e *GenericWorld[T, S]) RegisterComponents(component_ptr ...AnyComponentInstancesPtr) {
	var maxComponentId ComponentID

	for _, component := range component_ptr {
		if component.getId() > maxComponentId {
			maxComponentId = component.getId()
		}
	}

	e.components = make([]AnyComponentInstancesPtr, maxComponentId+1)

	for i := 0; i < len(component_ptr); i++ {
		component := component_ptr[i]
		component.registerComponentMask(e.entityComponentMask)
		e.components[component.getId()] = component
	}
}

func (e *GenericWorld[T, S]) RegisterUpdateSystems() *UpdateSystemBuilder[GenericWorld[T, S]] {
	return &UpdateSystemBuilder[GenericWorld[T, S]]{
		world:   e,
		systems: &e.updateSystems,
	}
}

func (e *GenericWorld[T, S]) RegisterDrawSystems() *DrawSystemBuilder[GenericWorld[T, S]] {
	return &DrawSystemBuilder[GenericWorld[T, S]]{
		ecs:     e,
		systems: &e.drawSystems,
	}
}

func (e *GenericWorld[T, S]) RunUpdateSystems() error {
	for i := range e.updateSystems {
		// If systems are sequantial, we dont spawn goroutines
		if len(e.updateSystems[i]) == 1 {
			e.updateSystems[i][0].Run(e)
			continue
		}

		e.wg.Add(len(e.updateSystems[i]))
		for j := range e.updateSystems[i] {
			// TODO prespawn goroutines for systems with MAX_N channels, where MAX_N is max number of parallel systems
			go func(system AnyUpdateSystem[GenericWorld[T, S]], e *GenericWorld[T, S]) {
				defer e.wg.Done()
				system.Run(e)
			}(e.updateSystems[i][j], e)
		}
		e.wg.Wait()
	}

	e.tick++
	e.Clean()

	return nil
}

func (e *GenericWorld[T, S]) RunDrawSystems(screen *ebiten.Image) {
	for i := range e.drawSystems {
		// If systems are sequantial, we dont spawn goroutines
		if len(e.drawSystems[i]) == 1 {
			e.drawSystems[i][0].Run(e, screen)
			continue
		}

		e.wg.Add(len(e.drawSystems[i]))
		for j := range e.drawSystems[i] {
			// TODO prespawn goroutines for systems with MAX_N channels, where MAX_N is max number of parallel systems
			go func(system AnyDrawSystem[GenericWorld[T, S]], e *GenericWorld[T, S], screen *ebiten.Image) {
				defer e.wg.Done()
				system.Run(e, screen)
			}(e.drawSystems[i][j], e, screen)
		}
		e.wg.Wait()
	}
}

func (e *GenericWorld[T, S]) CreateEntity(title string) EntityID {
	var newId = e.generateEntityID()
	e.entityComponentMask.Create(newId, ComponentBitArray256{})
	atomic.AddInt32(&e.size, 1)
	return newId
}

func (e *GenericWorld[T, S]) DestroyEntity(entityId EntityID) {
	mask := e.entityComponentMask.Get(entityId)

	// Entity should exist
	assert.NotNil(mask)

	for i := range mask.AllSet {
		e.components[i].Remove(entityId)
	}

	e.entityComponentMask.Remove(entityId)
	e.deletedEntityIDs = append(e.deletedEntityIDs, entityId)
	atomic.AddInt32(&e.size, -1)
}

func (e *GenericWorld[T, S]) Clean() {
	for i := range e.components {
		if e.components[i] == nil {
			continue
		}
		e.components[i].Clean()
	}
}

func (e *GenericWorld[T, S]) Size() int32 {
	return atomic.LoadInt32(&e.size)
}

func (e *GenericWorld[T, S]) generateEntityID() (newId EntityID) {
	if len(e.deletedEntityIDs) == 0 {
		newId = EntityID(atomic.AddInt32((*int32)(&e.lastEntityID), 1))
	} else {
		newId = e.deletedEntityIDs[len(e.deletedEntityIDs)-1]
		e.deletedEntityIDs = e.deletedEntityIDs[:len(e.deletedEntityIDs)-1]
	}
	return newId
}
