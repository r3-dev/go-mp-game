/*
This Source Code Form is subject to the terms of the Mozilla
Public License, v. 2.0. If a copy of the MPL was not distributed
with this file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

package main

import (
	"flag"
	"fmt"
	"gomp_game/pkgs/gomp/ecs"
	"image/color"
	"log"
	"os"
	"runtime/pprof"
	"strings"
	"sync"
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

const (
	width  = 2000
	height = 2000
)

type game struct {
	mx                  *sync.Mutex
	world               *ecs.World
	colorComponents     ecs.WorldComponents[color.RGBA]
	transformComponents ecs.WorldComponents[transform]

	imageBuffer     *ebiten.Image
	debugImage      *ebiten.Image
	screenBuffer    []byte
	scale           float64
	cursorPositionX int
	cursorPositionY int
	translateX      float64
	translateY      float64
}

func (g *game) Update() error {
	g.mx.Lock()
	defer g.mx.Unlock()

	_, dy := ebiten.Wheel()

	currentScale := g.scale
	g.scale += dy

	if g.scale < 0.1 {
		g.scale = 0.1
	} else if g.scale > 100 {
		g.scale = 100
	}

	isMouseButtonPressed := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)

	if isMouseButtonPressed {
		currentMouseX, currentMouseY := ebiten.CursorPosition()
		g.translateX += float64(currentMouseX - g.cursorPositionX)
		g.translateY += float64(currentMouseY - g.cursorPositionY)
		g.cursorPositionX = currentMouseX
		g.cursorPositionY = currentMouseY
	} else {
		g.cursorPositionX, g.cursorPositionY = ebiten.CursorPosition()
	}

	ratio := 1 - g.scale/currentScale
	g.translateX += (float64(g.cursorPositionX) - g.translateX) * ratio
	g.translateY += (float64(g.cursorPositionY) - g.translateY) * ratio

	g.world.RunSystems()

	g.screenBuffer = make([]byte, 4*width*height)
	g.colorComponents.AllParallel(func(entity ecs.EntityID, color *color.RGBA) bool {
		if color == nil {
			return true
		}

		transform := g.transformComponents.GetPtr(entity)
		if transform == nil {
			return true
		}

		index := (transform.x + transform.y*width) * 4
		*(*[4]byte)(unsafe.Pointer(&g.screenBuffer[index])) = *(*[4]byte)(unsafe.Pointer(color))
		return true
	})

	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	g.mx.Lock()
	defer g.mx.Unlock()

	screen.Clear()
	screen.Fill(color.RGBA{
		R: 49,
		G: 49,
		B: 49,
		A: 255,
	})
	g.debugImage.Clear()
	g.imageBuffer.Clear()

	g.imageBuffer.WritePixels(g.screenBuffer)

	var debugInfo = make([]string, 0)
	p := message.NewPrinter(language.Russian)

	debugInfo = append(debugInfo, fmt.Sprintf("TPS %0.2f", ebiten.ActualTPS()))
	debugInfo = append(debugInfo, fmt.Sprintf("FPS %0.2f", ebiten.ActualFPS()))
	debugInfo = append(debugInfo, fmt.Sprintf("Scale %0.2f", g.scale))
	debugInfo = append(debugInfo, p.Sprintf("Entity count %d", entityCount))
	ebitenutil.DebugPrint(g.debugImage, strings.Join(debugInfo, "\n"))

	op := new(ebiten.DrawImageOptions)
	op.GeoM.Scale(g.scale, g.scale)
	op.GeoM.Translate(g.translateX, g.translateY)
	screen.DrawImage(g.imageBuffer, op)

	op.GeoM.Reset()
	op.GeoM.Scale(2, 2)
	screen.DrawImage(g.debugImage, op)
}

func (g *game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	ebiten.SetRunnableOnUnfocused(true)
	ebiten.SetWindowSize(1000, 1000)
	ebiten.SetWindowTitle("1 mil pixel ecs")

	world := ecs.New("1 mil pixel")

	world.RegisterComponentTypes(
		&transformComponentType,
		&healthComponentType,
		&colorComponentType,
		&movementComponentType,
	)

	world.RegisterSystems().
		Parallel(
			new(spawnSystem),
		).
		Parallel(
			new(hpSystem),
		).
		Parallel(
			new(colorSystem),
		).
		Parallel(
			new(destroySystem),
		)

	newGame := game{
		mx:                  new(sync.Mutex),
		world:               &world,
		colorComponents:     colorComponentType.Instances(&world),
		transformComponents: transformComponentType.Instances(&world),
		imageBuffer:         ebiten.NewImage(width, height),
		debugImage:          ebiten.NewImage(250, 250),
		scale:               1,
	}

	newGame.screenBuffer = make([]byte, 4*width*height)

	if err := ebiten.RunGame(&newGame); err != nil {
		panic(err)
	}
}
