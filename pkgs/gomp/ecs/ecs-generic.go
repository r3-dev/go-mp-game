/*
This Source Code Form is subject to the terms of the Mozilla
Public License, v. 2.0. If a copy of the MPL was not distributed
with this file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

package ecs

import (
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2"
)

type GenericWorld[T any, S any] struct {
	Components *T
	Systems    *S

	components    []AnyComponentInstancesPtr
	updateSystems [][]AnyUpdateSystem[GenericWorld[T, S]]
	drawSystems   [][]AnyDrawSystem[GenericWorld[T, S]]

	deletedEntityIDs *PagedArray[EntityID]
	LastEntityID     EntityID

	tick int
	ID   ECSID

	wg   *sync.WaitGroup
	mx   *sync.Mutex
	size int32
}

func CreateGenericWorld[C any, US any](id ECSID, components *C, systems *US) GenericWorld[C, US] {
	ecs := GenericWorld[C, US]{
		ID:               id,
		Components:       components,
		Systems:          systems,
		wg:               new(sync.WaitGroup),
		mx:               new(sync.Mutex),
		deletedEntityIDs: NewPagedArray[EntityID](),
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

func (e *GenericWorld[T, S]) findComponentsFromStructRecursively(structValue reflect.Value, componentList []AnyComponentInstancesPtr, occupiedIds *big.Int) []AnyComponentInstancesPtr {
	compsType := structValue.Type()
	anyCompInstPtrType := reflect.TypeFor[AnyComponentInstancesPtr]()

	if occupiedIds == nil {
		occupiedIds = big.NewInt(0)
	}

	for i := range compsType.NumField() {
		if len(componentList) >= MAX_COMPONENTS_COUNT {
			panic("too many component types")
		}

		fld := compsType.Field(i)
		fldVal := structValue.FieldByIndex(fld.Index)

		// check for pointer and struct to ensure that type is instantiable
		if fld.Type.Kind() == reflect.Pointer && fld.Type.Elem().Kind() == reflect.Struct && fld.Type.Implements(anyCompInstPtrType) {
			var id ComponentID
			if idStr, ok := fld.Tag.Lookup("id"); !ok {
				panic(fmt.Sprintf("field %s in type %s doesn't have tag id", fld.Name, compsType.String()))
			} else if v, err := strconv.Atoi(idStr); err != nil {
				panic(fmt.Sprintf("field %s in type %s has invalid value \"%s\"", fld.Name, compsType.String(), idStr))
			} else if v < COMPONENT_ID_RANGE_LO || v > COMPONENT_ID_RANGE_HI {
				panic(fmt.Sprintf("field %s in type %s has id out of range (got %d, allowed [%d..%d])", fld.Name, compsType.String(), v, COMPONENT_ID_RANGE_LO, COMPONENT_ID_RANGE_HI))
			} else if occupiedIds.Bit(v) != 0 {
				panic(fmt.Sprintf("field %s in type %s has id conflict", fld.Name, compsType.String()))
			} else {
				id = ComponentID(v)
				occupiedIds = occupiedIds.SetBit(occupiedIds, v, 1)
			}

			ptr := reflect.New(fld.Type.Elem())
			fldVal.Set(ptr)
			ptr.Elem().FieldByName("ID").Set(reflect.ValueOf(id))
			ptr.MethodByName("Init").Call([]reflect.Value{})
			componentList = append(componentList, fldVal.Interface().(AnyComponentInstancesPtr))
		} else if fld.Anonymous && fld.Type.Kind() == reflect.Struct {
			componentList = e.findComponentsFromStructRecursively(fldVal, componentList, occupiedIds)
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

func (w *GenericWorld[T, S]) registerComponents(component_ptr ...AnyComponentInstancesPtr) {
	var maxComponentId ComponentID

	for _, component := range component_ptr {
		if component.getId() > maxComponentId {
			maxComponentId = component.getId()
		}
	}

	w.components = make([]AnyComponentInstancesPtr, maxComponentId+1)

	for i := 0; i < len(component_ptr); i++ {
		component := component_ptr[i]
		w.components[component.getId()] = component
	}
}

func (w *GenericWorld[T, S]) registerUpdateSystems() *UpdateSystemBuilder[GenericWorld[T, S]] {
	return &UpdateSystemBuilder[GenericWorld[T, S]]{
		world:   w,
		systems: &w.updateSystems,
	}
}

func (w *GenericWorld[T, S]) registerDrawSystems() *DrawSystemBuilder[GenericWorld[T, S]] {
	return &DrawSystemBuilder[GenericWorld[T, S]]{
		ecs:     w,
		systems: &w.drawSystems,
	}
}

func (w *GenericWorld[T, S]) RunUpdateSystems() error {
	for i := range w.updateSystems {
		// If systems are sequantial, we dont spawn goroutines
		if len(w.updateSystems[i]) == 1 {
			w.updateSystems[i][0].Run(w)
			continue
		}

		w.wg.Add(len(w.updateSystems[i]))
		for j := range w.updateSystems[i] {
			// TODO prespawn goroutines for systems with MAX_N channels, where MAX_N is max number of parallel systems
			go func(system AnyUpdateSystem[GenericWorld[T, S]], e *GenericWorld[T, S]) {
				defer e.wg.Done()
				system.Run(e)
			}(w.updateSystems[i][j], w)
		}
		w.wg.Wait()
	}

	w.tick++
	w.Clean()

	return nil
}

func (w *GenericWorld[T, S]) RunDrawSystems(screen *ebiten.Image) {
	for i := range w.drawSystems {
		// If systems are sequantial, we dont spawn goroutines
		if len(w.drawSystems[i]) == 1 {
			w.drawSystems[i][0].Run(w, screen)
			continue
		}

		w.wg.Add(len(w.drawSystems[i]))
		for j := range w.drawSystems[i] {
			// TODO prespawn goroutines for systems with MAX_N channels, where MAX_N is max number of parallel systems
			go func(system AnyDrawSystem[GenericWorld[T, S]], e *GenericWorld[T, S], screen *ebiten.Image) {
				defer e.wg.Done()
				system.Run(e, screen)
			}(w.drawSystems[i][j], w, screen)
		}
		w.wg.Wait()
	}
}

func (w *GenericWorld[T, S]) CreateEntity(title string) EntityID {
	w.mx.Lock()
	defer w.mx.Unlock()

	var newId = w.generateEntityID()
	// w.entityComponentMask.Create(newId, big.Int{})
	atomic.AddInt32(&w.size, 1)
	return newId
}

func (w *GenericWorld[T, S]) DestroyEntity(entityId EntityID) {
	w.mx.Lock()
	defer w.mx.Unlock()

	for _, component := range w.components {
		if component == nil {
			continue
		}

		if component.Has(entityId) {
			component.Remove(entityId)
		}
	}

	w.deletedEntityIDs.Append(entityId)
	atomic.AddInt32(&w.size, -1)
}

func (w *GenericWorld[T, S]) Clean() {
	for i := range w.components {
		if w.components[i] == nil {
			continue
		}
		w.components[i].Clean()
	}
	// e.deletedEntityIDs.Clean()
}

func (w *GenericWorld[T, S]) Size() int32 {
	return atomic.LoadInt32(&w.size)
}

func (w *GenericWorld[T, S]) generateEntityID() (newId EntityID) {
	if w.deletedEntityIDs.Len() == 0 {
		newId = EntityID(atomic.AddInt32((*int32)(&w.LastEntityID), 1))
	} else {
		newId = *w.deletedEntityIDs.Last()
		w.deletedEntityIDs.SoftReduce()
	}
	return newId
}
