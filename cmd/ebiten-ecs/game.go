/*
This Source Code Form is subject to the terms of the Mozilla
Public License, v. 2.0. If a copy of the MPL was not distributed
with this file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

package main

import (
	"gomp_game/pkgs/gomp/ecs"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

type ClientWorld = ecs.GenericWorld[clientComponents, clientSystems]

type client struct {
	world *ClientWorld
}

type sharedComponents struct {
	Destroy   *ecs.ComponentManager[destroy]
	Transform *ecs.ComponentManager[transform]
	Health    *ecs.ComponentManager[health]
}

type sharedSystems struct {
	Spawn   *systemSpawn
	CalcHp  *systemCalcHp
	Destroy *systemDestroyRemovedEntities
}

type clientComponents struct {
	sharedComponents

	Color  *ecs.ComponentManager[color.RGBA]
	Camera *ecs.ComponentManager[camera]
}

type clientSystems struct {
	sharedSystems

	CalcCol *systemCalcColor
	Draw    *systemDraw
}

func newGameClient() (c client) {
	// Create world and register components and systems
	world := ecs.CreateGenericWorld(0, new(clientComponents), new(clientSystems))

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
