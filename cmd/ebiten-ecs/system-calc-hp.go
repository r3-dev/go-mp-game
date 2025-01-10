/*
This Source Code Form is subject to the terms of the Mozilla
Public License, v. 2.0. If a copy of the MPL was not distributed
with this file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

package main

import (
	"gomp_game/pkgs/gomp/ecs"
)

type systemCalcHp struct{}

func (s *systemCalcHp) Init(world *ClientWorld) {}
func (s *systemCalcHp) Run(world *ClientWorld) {
	world.Components.Health.AllParallel(func(entity ecs.EntityID, h *health) bool {
		h.hp--

		if h.hp <= 0 {
			world.Components.Destroy.Create(entity, destroy{})
		}

		return true
	})
}
func (s *systemCalcHp) Destroy(world *ClientWorld) {}
