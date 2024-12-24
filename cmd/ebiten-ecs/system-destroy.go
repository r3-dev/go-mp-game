/*
This Source Code Form is subject to the terms of the Mozilla
Public License, v. 2.0. If a copy of the MPL was not distributed
with this file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

package main

import (
	"gomp_game/pkgs/gomp/ecs"
)

type systemDestroyRemovedEntities struct {
	n int
}

func (s *systemDestroyRemovedEntities) Init(world *ClientWorld) {}
func (s *systemDestroyRemovedEntities) Run(world *ClientWorld) {
	s.n = 0
	world.Components.Destroy.All(func(e ecs.EntityID, h *destroy) bool {
		world.DestroyEntity(e)
		return true
	})
}
func (s *systemDestroyRemovedEntities) Destroy(world *ClientWorld) {}
