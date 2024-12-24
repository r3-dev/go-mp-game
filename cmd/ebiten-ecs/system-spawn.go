/*
This Source Code Form is subject to the terms of the Mozilla
Public License, v. 2.0. If a copy of the MPL was not distributed
with this file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

package main

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"os"
	"runtime/pprof"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type systemSpawn struct {
	pprofEnabled bool
}

const (
	minHpPercentage = 20
	minMaxHp        = 500
	maxMaxHp        = 2000
)

func (s *systemSpawn) Init(world *ClientWorld) {}
func (s *systemSpawn) Run(world *ClientWorld) {
	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		for range rand.Intn(1000) {

			newCreature := world.CreateEntity("Creature")

			t := transform{
				x: rand.Int31n(1000),
				y: rand.Int31n(1000),
			}

			world.Components.Transform.Create(newCreature, t)

			maxHp := minMaxHp + rand.Int31n(maxMaxHp-minMaxHp)
			hp := int32(float32(maxHp) * float32(minHpPercentage+rand.Int31n(100-minHpPercentage)) / 100)

			h := health{
				hp:    hp,
				maxHp: maxHp,
			}

			world.Components.Health.Create(newCreature, h)

			c := color.RGBA{
				R: 0,
				G: 0,
				B: 0,
				A: 0,
			}

			world.Components.Color.Create(newCreature, c)
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF9) {
		if *cpuprofile != "" {
			if s.pprofEnabled {
				pprof.StopCPUProfile()
				fmt.Println("CPU Profile Stopped")
			} else {
				f, err := os.Create(*cpuprofile)
				if err != nil {
					log.Fatal(err)
				}
				pprof.StartCPUProfile(f)
				fmt.Println("CPU Profile Started")
			}

			s.pprofEnabled = !s.pprofEnabled
		}
	}
}
func (s *systemSpawn) Destroy(world *ClientWorld) {}
