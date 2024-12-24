/*
This Source Code Form is subject to the terms of the Mozilla
Public License, v. 2.0. If a copy of the MPL was not distributed
with this file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"gomp_game/pkgs/gomp/ecs"
	"image/color"
	"reflect"
)

type ClientWorld = ecs.GenericWorld[clientComponents, clientSystems]

type client struct {
	world *ClientWorld
}

type clientComponents struct {
	Destroy   *ecs.ComponentManager[destroy]
	Camera    *ecs.ComponentManager[camera]
	Transform *ecs.ComponentManager[transform]
	Health    *ecs.ComponentManager[health]
	Color     *ecs.ComponentManager[color.RGBA]
}

type clientSystems struct {
	Spawn   *systemSpawn
	CalcHp  *systemCalcHp
	CalcCol *systemCalcColor
	Destroy *systemDestroyRemovedEntities
	Draw    *systemDraw
}

func newGameClient() (c client) {
	// Create component managers
	components := clientComponents{
		Color:     ecs.CreateComponentManager[color.RGBA](COLOR_COMPONENT_ID),
		Camera:    ecs.CreateComponentManager[camera](CAMERA_COMPONENT_ID),
		Health:    ecs.CreateComponentManager[health](HEALTH_COMPONENT_ID),
		Destroy:   ecs.CreateComponentManager[destroy](DESTROY_COMPONENT_ID),
		Transform: ecs.CreateComponentManager[transform](TRANSFORM_COMPONENT_ID),
	}

	// Create systems
	//systems := clientSystems{
	//	Spawn:   new(systemSpawn),
	//	CalcHp:  new(systemCalcHp),
	//	CalcCol: new(systemCalcColor),
	//	Destroy: new(systemDestroyRemovedEntities),
	//	Draw:    new(systemDraw),
	//}
	systems := clientSystems{}
	valueOfSystems := reflect.ValueOf(&systems)
	sListLen := valueOfSystems.Elem().NumField()
	for i := 0; i < sListLen; i++ {
		ptr := reflect.New(valueOfSystems.Elem().Field(i).Type().Elem())
		valueOfSystems.Elem().Field(i).Set(ptr)
	}

	// Create world
	world := ecs.CreateGenericWorld(0, &components, &systems)

	// Register components
	//world.RegisterComponents(
	//	components.color,
	//	components.camera,
	//	components.health,
	//	components.transform,
	//	components.destroy,
	//)
	valueOfComponents := reflect.ValueOf(components)
	cListLen := valueOfComponents.NumField()
	componentList := make([]ecs.AnyComponentInstancesPtr, 0, cListLen)
	for i := 0; i < cListLen; i++ {
		componentList = append(componentList, valueOfComponents.Field(i).Interface().(ecs.AnyComponentInstancesPtr))
	}
	world.RegisterComponents(
		componentList...,
	)

	// Register update systems
	world.RegisterUpdateSystems().
		Sequential(
			systems.Spawn,
			systems.CalcHp,
			systems.CalcCol,
			systems.Destroy,
		)

	// Register draw systems
	world.RegisterDrawSystems().
		Sequential(
			systems.Draw,
		)

	newClient := client{
		world: &world,
	}

	return newClient
}

func (c *client) Update() error {
	return c.world.RunUpdateSystems()
}

func (c *client) Draw(screen *ebiten.Image) {
	c.world.RunDrawSystems(screen)
}

func (c *client) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}
