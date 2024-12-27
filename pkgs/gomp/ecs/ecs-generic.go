/*
This Source Code Form is subject to the terms of the Mozilla
Public License, v. 2.0. If a copy of the MPL was not distributed
with this file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

package ecs

import (
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/negrel/assert"
)

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

func CreateGenericWorld[C any, US any](id ECSID, components *C, systems *US) GenericWorld[C, US] {
	maskSet := CreateComponentManager[ComponentBitArray256]()
	maskSet.ID = ENTITY_COMPONENT_MASK_ID
	ecs := GenericWorld[C, US]{
		ID:                  id,
		Components:          components,
		Systems:             systems,
		wg:                  new(sync.WaitGroup),
		deletedEntityIDs:    make([]EntityID, 0, PREALLOC_DELETED_ENTITIES),
		entityComponentMask: maskSet,
	}

	// Register components
	ecs.registerComponents(
		ecs.findComponentsFromStructRecursively(reflect.ValueOf(components).Elem(), nil, nil)...,
	)

	// Register systems
	updSystems, drawSystems := ecs.findSystemsFromStructRecursively(reflect.ValueOf(systems).Elem(), nil, nil)
	ecs.registerUpdateSystems().Sequential(updSystems...)
	ecs.registerDrawSystems().Sequential(drawSystems...)

	return ecs
}

func (e *GenericWorld[T, S]) findComponentsFromStructRecursively(structValue reflect.Value, componentList []AnyComponentInstancesPtr, freeCompId *ComponentID) []AnyComponentInstancesPtr {
	compsType := structValue.Type()
	anyCompInstPtrType := reflect.TypeFor[AnyComponentInstancesPtr]()

	var compId ComponentID = 0
	if freeCompId == nil {
		freeCompId = &compId
	}

	for i := range compsType.NumField() {
		if *freeCompId == MAX_COMPONENTS_COUNT {
			panic("too many component types")
		}

		fld := compsType.Field(i)
		fldVal := structValue.FieldByIndex(fld.Index)

		// check for pointer and struct to ensure that type is instantiable
		if fld.Type.Kind() == reflect.Pointer && fld.Type.Elem().Kind() == reflect.Struct && fld.Type.Implements(anyCompInstPtrType) {
			ptr := reflect.New(fld.Type.Elem())
			fldVal.Set(ptr)
			ptr.Elem().FieldByName("ID").Set(reflect.ValueOf(*freeCompId))
			ptr.MethodByName("Init").Call([]reflect.Value{})
			*freeCompId++
			componentList = append(componentList, fldVal.Interface().(AnyComponentInstancesPtr))
		} else if fld.Anonymous && fld.Type.Kind() == reflect.Struct {
			componentList = e.findComponentsFromStructRecursively(fldVal, componentList, freeCompId)
		}
	}

	return componentList
}

func (e *GenericWorld[T, S]) findSystemsFromStructRecursively(
	structValue reflect.Value,
	systemUpdList []AnyUpdateSystem[GenericWorld[T, S]],
	systemDrawList []AnyDrawSystem[GenericWorld[T, S]],
) (updSystems []AnyUpdateSystem[GenericWorld[T, S]], drawSystems []AnyDrawSystem[GenericWorld[T, S]]) {
	sysType := structValue.Type()
	anyUpdateSysType := reflect.TypeFor[AnyUpdateSystem[GenericWorld[T, S]]]()
	anyDrawSysType := reflect.TypeFor[AnyDrawSystem[GenericWorld[T, S]]]()

	for i := range sysType.NumField() {
		fld := sysType.Field(i)
		fldVal := structValue.FieldByIndex(fld.Index)

		if fld.Anonymous && fld.Type.Kind() == reflect.Struct {
			systemUpdList, systemDrawList = e.findSystemsFromStructRecursively(fldVal, systemUpdList, systemDrawList)
		} else if fld.Type.Kind() == reflect.Pointer {
			if fld.Type.Implements(anyUpdateSysType) {
				ptr := reflect.New(fld.Type.Elem())
				fldVal.Set(ptr)
				systemUpdList = append(systemUpdList, ptr.Interface().(AnyUpdateSystem[GenericWorld[T, S]]))
			} else if fld.Type.Implements(anyDrawSysType) {
				ptr := reflect.New(fld.Type.Elem())
				fldVal.Set(ptr)
				systemDrawList = append(systemDrawList, ptr.Interface().(AnyDrawSystem[GenericWorld[T, S]]))
			}
		}
	}

	return systemUpdList, systemDrawList
}

func (e *GenericWorld[T, S]) registerComponents(component_ptr ...AnyComponentInstancesPtr) {
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

func (e *GenericWorld[T, S]) registerUpdateSystems() *UpdateSystemBuilder[GenericWorld[T, S]] {
	return &UpdateSystemBuilder[GenericWorld[T, S]]{
		world:   e,
		systems: &e.updateSystems,
	}
}

func (e *GenericWorld[T, S]) registerDrawSystems() *DrawSystemBuilder[GenericWorld[T, S]] {
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
