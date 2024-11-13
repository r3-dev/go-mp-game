/*
This Source Code Form is subject to the terms of the Mozilla
Public License, v. 2.0. If a copy of the MPL was not distributed
with this file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

package systems

import (
	"gomp_game/pkgs/example/components"
	"gomp_game/pkgs/gomp"
	"log"
	"math"

	"github.com/yohamta/donburi"
)

var HeroMoveSystem = gomp.CreateSystem(new(heroMoveController))

type heroMoveController struct {
	world donburi.World
}

func (c *heroMoveController) Init(world donburi.World) {
	c.world = world
}

func (c *heroMoveController) Update(dt float64) {
	components.HeroIntentComponent.Query.Each(c.world, func(e *donburi.Entry) {
		body := gomp.BodyComponent.Query.Get(e)
		if body == nil {
			return
		}

		intent := components.HeroIntentComponent.Query.Get(e)

		playerSpeed := 200.0
		newX := intent.Move.X * playerSpeed
		newY := -intent.Move.Y * playerSpeed

		log.Println(math.Sqrt(newX*newX + newY*newY))

		body.SetVelocity(newX, newY)
	})
}
