/*
This Source Code Form is subject to the terms of the Mozilla
Public License, v. 2.0. If a copy of the MPL was not distributed
with this file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

package ecs

import (
	"iter"
	"math/bits"
)

type ChunkArray[T any] struct {
	buffer               []ChunkArrayElement[T]
	current              *ChunkArrayElement[T]
	last                 *ChunkArrayElement[T]
	size                 int
	initialChunkCapPower int
	initialBufferCap     int
	chunkCapPower        int
	bufferCapPower       int
	bufferSizeIndex      int
}

func NewChunkArray[T any](bufferCapacityPower int, chunkCapacityPower int) (arr *ChunkArray[T]) {
	arr = new(ChunkArray[T])
	arr.initialBufferCap = 1 << bufferCapacityPower
	arr.initialChunkCapPower = chunkCapacityPower

	arr.bufferCapPower = bufferCapacityPower
	arr.chunkCapPower = chunkCapacityPower

	arr.buffer = make([]ChunkArrayElement[T], 1<<bufferCapacityPower)
	arr.bufferSizeIndex = 0

	chunk := arr.makeChunk()
	chunk.parent = arr

	arr.current = chunk
	arr.last = chunk

	return arr
}

func (a *ChunkArray[T]) Len() int {
	return a.size
}

func (a *ChunkArray[T]) Get(index int) *T {
	pageIndex := a.getPageIdByIndex(index)
	page := &a.buffer[pageIndex]

	index -= page.startingIndex

	return &(page.data[index])
}

func (a *ChunkArray[T]) Set(index int, value T) (result *T, ok bool) {
	pageIndex := a.getPageIdByIndex(index)
	page := a.buffer[pageIndex]

	index -= page.startingIndex

	page.data[index] = value

	return &page.data[index], true
}

func (a *ChunkArray[T]) Append(value T) (int, *T) {
	var index, result = a.current.Append(value)
	index = a.size
	a.size++
	return index, result
}

func (a *ChunkArray[T]) SoftReduce() {
	a.current.SoftReduce()
}

func (a *ChunkArray[T]) Clean() {
	a.last.Clean()
}

func (a *ChunkArray[T]) Swap(i, j int) {
	x := *a.Get(i)
	y := *a.Get(j)

	a.Set(j, x)
	a.Set(i, y)
}

func (a *ChunkArray[T]) Last() (index int, value T, ok bool) {
	var last = a.last
	index = last.size - 1
	if index < 0 {
		if a.last.prev != nil {
			a.last = a.last.prev
			return a.Last()
		}

		return -1, value, false
	}

	return index + last.startingIndex, last.data[index], true
}

func (a *ChunkArray[T]) extendBuffer() {
	a.bufferCapPower++
	a.buffer = append(a.buffer, make([]ChunkArrayElement[T], 1<<a.bufferCapPower)...)
}

func (a *ChunkArray[T]) makeChunk() *ChunkArrayElement[T] {
	if a.bufferSizeIndex >= len(a.buffer) {
		a.extendBuffer()
	}

	chunk := &a.buffer[a.bufferSizeIndex]
	chunk.parent = a
	chunk.data = make([]T, 0, 1<<a.chunkCapPower)
	chunk.startingIndex = ((1<<a.bufferSizeIndex - 1) << a.initialChunkCapPower)
	a.chunkCapPower++
	a.bufferSizeIndex++

	a.current = chunk
	a.last = chunk

	return chunk
}

func (a *ChunkArray[T]) getPageIdByIndex(index int) int {
	return bits.Len64(uint64(index>>a.initialChunkCapPower+1)) - 1
}

func (a *ChunkArray[T]) Iter() iter.Seq2[int, *T] {
	return func(yield func(int, *T) bool) {
		for i := range a.buffer {
			var chunk = &a.buffer[i]
			var offset = ((1<<i - 1) << a.initialChunkCapPower)

			for j := range chunk.data {
				if !yield(offset+j, &chunk.data[j]) {
					return
				}
			}
		}
	}
}

// ======
// ======
// ======

type ChunkArrayElement[T any] struct {
	data          []T
	next          *ChunkArrayElement[T]
	prev          *ChunkArrayElement[T]
	parent        *ChunkArray[T]
	startingIndex int
	size          int
}

func (c *ChunkArrayElement[T]) Get(index int) (data T, ok bool) {
	data = c.data[index]
	return data, true
}

func (c *ChunkArrayElement[T]) Set(index int, value T) (*T, bool) {
	c.data[index] = value
	return &c.data[index], true
}

func (c *ChunkArrayElement[T]) Append(value T) (index int, result *T) {
	index = c.size

	if index < len(c.data) {
		c.data[index] = value
		c.size++
		return index, &c.data[index]
	}

	if index < cap(c.data) {
		c.data = append(c.data, value)
		c.size++
		return index, &c.data[index]
	}

	if c.next == nil {
		chunk := c.parent.makeChunk()
		chunk.prev = c
		c.next = chunk
	}

	return c.next.Append(value)
}

func (c *ChunkArrayElement[T]) SoftReduce() {
	if c.size > 0 {
		c.size--
		c.parent.size--
		return
	}

	if c.prev == nil {
		return
	}

	c.parent.current = c.prev
	c.prev.SoftReduce()
}

func (c *ChunkArrayElement[T]) Clean() {
	c.data = c.data[:c.size]

	if len(c.data) == 0 {
		if c.next != nil {
			c.parent.last = c
			c.next = nil
		}

		if c.prev != nil {
			c.prev.Clean()
		}
	}
}
