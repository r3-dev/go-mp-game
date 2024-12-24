/*
This Source Code Form is subject to the terms of the Mozilla
Public License, v. 2.0. If a copy of the MPL was not distributed
with this file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

package main

import (
	"gomp_game/pkgs/gomp/ecs"
	"image/color"
)

type systemCalcColor struct {
	baseColor color.RGBA
}

func (s *systemCalcColor) Init(world *ClientWorld) {
	s.baseColor = color.RGBA{25, 220, 200, 255}
}
func (s *systemCalcColor) Run(world *ClientWorld) {
	components := world.Components

	components.Color.AllParallel(func(ei ecs.EntityID, c *color.RGBA) bool {
		health := components.Health.Get(ei)
		if health == nil {
			return true
		}

		hpPercentage := float32(health.hp) / float32(health.maxHp)

		c.R = uint8(hpPercentage * float32(s.baseColor.R))
		c.G = uint8(hpPercentage * float32(s.baseColor.G))
		c.B = uint8(hpPercentage * float32(s.baseColor.B))
		c.A = s.baseColor.A

		return true
	})
}
func (s *systemCalcColor) Destroy(world *ClientWorld) {}
