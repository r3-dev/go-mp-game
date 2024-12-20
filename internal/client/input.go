/*
This Source Code Form is subject to the terms of the Mozilla
Public License, v. 2.0. If a copy of the MPL was not distributed
with this file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

package client

import input "github.com/quasilyte/ebitengine-input"

const (
	ActionMoveLeft input.Action = iota
	ActionMoveRight
	ActionMoveUp
	ActionMoveDown
)

type Inputs struct {
	System   input.System
	keyMaps  map[uint8]*input.Keymap
	Handlers map[uint8]*input.Handler
}

func NewInputs(availableDevices input.DeviceKind) *Inputs {
	inputs := &Inputs{
		keyMaps:  make(map[uint8]*input.Keymap),
		Handlers: make(map[uint8]*input.Handler),
	}

	inputs.System.Init(input.SystemConfig{
		DevicesEnabled: input.AnyDevice,
	})

	defaultKeyMap := &input.Keymap{
		ActionMoveLeft:  {input.KeyGamepadLeft, input.KeyLeft, input.KeyA},
		ActionMoveRight: {input.KeyGamepadRight, input.KeyRight, input.KeyD},
		ActionMoveUp:    {input.KeyGamepadUp, input.KeyUp, input.KeyW},
		ActionMoveDown:  {input.KeyGamepadDown, input.KeyDown, input.KeyS},
	}

	inputs.Register(0, defaultKeyMap)

	return inputs
}

func (i *Inputs) Register(deviceId uint8, keyMap *input.Keymap) {
	handler := i.System.NewHandler(deviceId, *keyMap)

	i.keyMaps[deviceId] = keyMap
	i.Handlers[deviceId] = handler
}
