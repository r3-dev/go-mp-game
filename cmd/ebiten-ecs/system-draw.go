/*
This Source Code Form is subject to the terms of the Mozilla
Public License, v. 2.0. If a copy of the MPL was not distributed
with this file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

package main

import (
	"fmt"
	"gomp_game/pkgs/gomp/ecs"
	"image"
	"image/color"
	"image/draw"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/negrel/assert"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type systemDraw struct {
	buffer    *image.RGBA
	debugInfo []string
	p         *message.Printer
}

func (s *systemDraw) Init(world *ClientWorld) {
	s.p = message.NewPrinter(language.Russian)

	newcamera := world.CreateEntity("Camera")
	world.Components.Camera.Create(newcamera, camera{
		mainLayer: cameraLayer{
			image: ebiten.NewImage(width, height),
			zoom:  1,
		},
		debugLayer: cameraLayer{
			image: ebiten.NewImage(width, height),
			zoom:  2,
		},
	})

	s.buffer = image.NewRGBA(image.Rect(0, 0, width, height))
	s.debugInfo = make([]string, 0)
}

func (s *systemDraw) Run(world *ClientWorld, screen *ebiten.Image) {
	components := world.Components
	systems := world.Systems
	_, dy := ebiten.Wheel()

	draw.Draw(s.buffer, s.buffer.Bounds(), &image.Uniform{color.Transparent}, image.Point{}, draw.Src)

	components.Color.AllParallel(func(entity ecs.EntityID, color *color.RGBA) bool {
		if color == nil {
			return true
		}

		transform := components.Transform.Get(entity)

		s.buffer.SetRGBA(int(transform.x), int(transform.y), *color)
		return true
	})

	var mainCamera *camera
	components.Camera.AllData(func(c *camera) bool {
		mainCamera = c
		return false
	})
	assert.True(mainCamera != nil, "No Camera found")

	mainCamera.mainLayer.image.Clear()
	mainCamera.debugLayer.image.Clear()

	mainCamera.mainLayer.zoom += float64(dy)
	mainCamera.mainLayer.image.WritePixels(s.buffer.Pix)

	if mainCamera.mainLayer.zoom < 0.1 {
		mainCamera.mainLayer.zoom = 0.1
	} else if mainCamera.mainLayer.zoom > 100 {
		mainCamera.mainLayer.zoom = 100
	}

	s.debugInfo = append(s.debugInfo, fmt.Sprintf("TPS %0.2f", ebiten.ActualTPS()))
	s.debugInfo = append(s.debugInfo, fmt.Sprintf("FPS %0.2f", ebiten.ActualFPS()))
	s.debugInfo = append(s.debugInfo, fmt.Sprintf("Zoom %0.2f", mainCamera.mainLayer.zoom))
	s.debugInfo = append(s.debugInfo, s.p.Sprintf("Entity count %d", world.Size()))
	s.debugInfo = append(s.debugInfo, s.p.Sprintf("Transforms count %d", components.Transform.Len()))
	s.debugInfo = append(s.debugInfo, s.p.Sprintf("Healths count %d", components.Health.Len()))
	s.debugInfo = append(s.debugInfo, s.p.Sprintf("Colors count %d", components.Color.Len()))
	s.debugInfo = append(s.debugInfo, s.p.Sprintf("Cameras count %d", components.Camera.Len()))
	s.debugInfo = append(s.debugInfo, s.p.Sprintf("Destroys count %d", components.Destroy.Len()))
	s.debugInfo = append(s.debugInfo, s.p.Sprintf("Pprof %d", systems.Spawn.pprofEnabled))

	ebitenutil.DebugPrint(mainCamera.debugLayer.image, strings.Join(s.debugInfo, "\n"))
	s.debugInfo = s.debugInfo[:0]

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Reset()
	op.GeoM.Scale(mainCamera.mainLayer.zoom, mainCamera.mainLayer.zoom)
	screen.DrawImage(mainCamera.mainLayer.image, op)

	op.GeoM.Reset()
	op.GeoM.Scale(mainCamera.debugLayer.zoom, mainCamera.debugLayer.zoom)
	screen.DrawImage(mainCamera.debugLayer.image, op)
}
func (s *systemDraw) Destroy(world *ClientWorld) {}
