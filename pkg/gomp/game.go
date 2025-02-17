/*
This Source Code Form is subject to the terms of the Mozilla
Public License, v. 2.0. If a copy of the MPL was not distributed
with this file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

package gomp

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/yohamta/donburi"
)

type Game struct {
	mx sync.Mutex
	wg *sync.WaitGroup

	World        donburi.World
	systems      []System
	LoadedScenes map[string]*Scene
	Network      *Network

	tickRate time.Duration
	Debug    bool
}

func (g *Game) Init(tickRate time.Duration) {
	g.World = donburi.NewWorld()
	// g.systems = []ecs.System{}
	g.tickRate = tickRate
	g.wg = new(sync.WaitGroup)
	g.LoadedScenes = make(map[string]*Scene)
	g.Debug = false
}

func (g *Game) Run(ctx context.Context) {
	ticker := time.NewTicker(g.tickRate)
	defer ticker.Stop()

	dt := g.tickRate.Seconds()

	g.Update(dt)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			g.Update(dt)
		}
	}
}

func (g *Game) Update(dt float64) {
	g.mx.Lock()
	defer g.mx.Unlock()

	if g.Debug {
		log.Println("=========Game UPDATE START==========")
		defer log.Println("=========Game UPDATE FINISH=========")
		log.Println("dt:", dt)
	}

	dtTreshold := g.tickRate.Seconds() * 2

	if dt > dtTreshold {
		log.Println("WARNING: Game tick rate is too high", dt, dtTreshold)
		return
	}

	lenSys := len(g.systems)
	for i := 0; i < lenSys; i++ {
		g.systems[i].Update(dt)
	}

	g.wg.Add(len(g.LoadedScenes))
	for i := range g.LoadedScenes {
		go updateSystemsAsync(g.LoadedScenes[i], dt, g.wg)
	}
	g.wg.Wait()
}

func (g *Game) LoadScene(scene Scene) *Scene {
	g.mx.Lock()
	defer g.mx.Unlock()

	if g.Debug {
		log.Println("Loading scene:", scene.Name)
		defer log.Println("Scene loaded:", scene.Name)
	}

	//check if scene already exists
	suffix := 1

	for {
		prefixedName := fmt.Sprintf("%s_%d", scene.Name, suffix)

		if _, ok := g.LoadedScenes[prefixedName]; ok {
			suffix++
			continue
		}

		scene.Name = prefixedName
		break
	}

	c := scene.SceneComponent.New(SceneData{Name: scene.Name})
	entitiesLen := len(scene.Entities)
	for i := 0; i < entitiesLen; i++ {
		scene.Entities[i](g.World, c)
	}

	g.LoadedScenes[scene.Name] = &scene
	return &scene
}

func (g *Game) UnloadScene(scene *Scene) {
	g.mx.Lock()
	defer g.mx.Unlock()

	if scene == nil {
		panic("Trying to unload nil scene")
	}

	if g.Debug {
		log.Println("Unloading scene: ", scene.Name)
		defer log.Println("Scene unloaded: ", scene.Name)
	}

	// check if scene exists
	if _, ok := g.LoadedScenes[scene.Name]; !ok {
		return
	}

	scene.SceneComponent.Query.Each(g.World, func(e *donburi.Entry) {
		g.World.Remove(e.Entity())
	})

	delete(g.LoadedScenes, scene.Name)
}

func (g *Game) UnloadAllScenes() {
	for i := range g.LoadedScenes {
		g.UnloadScene(g.LoadedScenes[i])
	}
}

func (g *Game) RegisterSystems(systems ...System) {
	g.mx.Lock()
	defer g.mx.Unlock()

	for i := range systems {
		g.systems = append(g.systems, systems[i])
		g.systems[i].Init(g)
	}
}

func updateSystemsAsync(scene *Scene, dt float64, wg *sync.WaitGroup) {
	defer wg.Done()
	lenSys := len(scene.Systems)
	for i := 0; i < lenSys; i++ {
		scene.Systems[i].Update(dt)
	}
}
