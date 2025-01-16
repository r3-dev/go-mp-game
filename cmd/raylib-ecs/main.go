/*
This Source Code Form is subject to the terms of the Mozilla
Public License, v. 2.0. If a copy of the MPL was not distributed
with this file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

package main

import (
	"gomp_game/cmd/raylib-ecs/components"
	"gomp_game/cmd/raylib-ecs/systems"
	"gomp_game/pkgs/gomp/ecs"
)

func main() {
	world := ecs.CreateWorld("main-raylib")
	defer world.Destroy()

	// patch := world.GetPatch()
	// world.ApplyPatch(patch)

	// components.TransformService.OnPatch(func(newstate components.Transform, oldstate components.Transform) components.Transform {
	// 	return new
	// })

	world.RegisterComponentServices(
		&components.TransformService,
		&components.HealthService,
		&components.ColorService,
	)

	world.RegisterSystems().
		Parallel(
			&systems.SpawnService,
			&systems.HpService,
			&systems.ColorService,
		).
		Sequential(
			&systems.RenderService,
		)

	world.Run(60)
}
