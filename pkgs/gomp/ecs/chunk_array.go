/*
This Source Code Form is subject to the terms of the Mozilla
Public License, v. 2.0. If a copy of the MPL was not distributed
with this file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

package ecs

import (
	"math/bits"
	"runtime"
	"sync"
)

type ChunkArray[T any] struct {
	buffer               []ChunkArrayElement[T]
	current              *ChunkArrayElement[T]
	size                 int
	bufferSize           int
	initialChunkCapPower int
	initialBufferCap     int
	chunkCapPower        int
	bufferCapPower       int
	numBufferAllocations int
	parallelCount        uint
}

func NewChunkArray[T any](bufferCapacityPower int, chunkCapacityPower int) (arr *ChunkArray[T]) {
	arr = new(ChunkArray[T])
	arr.initialBufferCap = 1 << bufferCapacityPower
	arr.initialChunkCapPower = chunkCapacityPower

	arr.bufferCapPower = bufferCapacityPower
	arr.chunkCapPower = chunkCapacityPower

	arr.buffer = make([]ChunkArrayElement[T], 1<<bufferCapacityPower)
	arr.numBufferAllocations = 0
	arr.bufferSize = 1

	chunk := arr.makeChunk()
	chunk.parent = arr

	arr.current = chunk
	arr.parallelCount = uint(runtime.NumCPU()) / 2

	return arr
}

func (a *ChunkArray[T]) Len() int {
	return a.size
}

func (a *ChunkArray[T]) Get(index int) *T {
	if index >= a.size {
		return nil
	}

	pageIndex := a.getPageIdByIndex(index)
	page := &a.buffer[pageIndex]

	index -= page.startingIndex

	return &(page.data[index])
}

func (a *ChunkArray[T]) Set(index int, value T) (result *T, ok bool) {
	if index >= a.size {
		return nil, false
	}

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
	if a.current.size > 0 {
		a.current.size--
		a.size--
		return
	}

	a.bufferSize--
	a.current = a.current.prev
	a.SoftReduce()
}

func (a *ChunkArray[T]) Clean() {
	for i := len(a.buffer) - 1; i >= 0; i-- {
		chunk := &a.buffer[i]
		if chunk.size == len(chunk.data) {
			continue
		}

		chunk.Clean()
	}
}

func (a *ChunkArray[T]) Copy(fromIndex, toIndex int) {
	from := a.Get(fromIndex)
	to := a.Get(toIndex)
	*to = *from
}

func (a *ChunkArray[T]) Swap(i, j int) (newI, NewJ *T) {
	x := a.Get(i)
	y := a.Get(j)
	*x, *y = *y, *x
	return x, y
}

// func (a *ChunkArray[T]) Last() (index int, value T, ok bool) {
// 	var last = a.last
// 	index = last.size - 1
// 	if index < 0 {
// 		if a.last.prev != nil {
// 			a.last = a.last.prev
// 			return a.Last()
// 		}

// 		return -1, value, false
// 	}

// 	return index + last.startingIndex, last.data[index], true
// }

func (a *ChunkArray[T]) Last() (index int, value T, ok bool) {
	index = a.size - 1
	if index < 0 {
		return -1, value, false
	}

	return index, *a.Get(index), true
}

func (a *ChunkArray[T]) extendBuffer() {
	a.bufferCapPower++
	a.buffer = append(a.buffer, make([]ChunkArrayElement[T], 1<<a.bufferCapPower)...)
}

func (a *ChunkArray[T]) makeChunk() *ChunkArrayElement[T] {
	if a.numBufferAllocations >= len(a.buffer) {
		a.extendBuffer()
	}

	chunk := &a.buffer[a.numBufferAllocations]
	chunk.parent = a
	chunk.data = make([]T, 0, 1<<a.chunkCapPower)
	chunk.startingIndex = ((1<<a.numBufferAllocations - 1) << a.initialChunkCapPower)
	a.chunkCapPower++
	a.numBufferAllocations++

	a.current = chunk

	return chunk
}

func (a *ChunkArray[T]) getPageIdByIndex(index int) int {
	return bits.Len64(uint64(index>>a.initialChunkCapPower+1)) - 1
}

type ChunkArrayIndex struct {
	local        int
	globalOffset int
	page         int
}

func (a *ChunkArray[T]) All(yield func(ChunkArrayIndex, *T) bool) {
	var chunk *ChunkArrayElement[T]
	var data []T
	var index ChunkArrayIndex

	buffer := a.buffer

	for i := a.bufferSize - 1; i >= 0; i-- {
		chunk = &buffer[i]
		index.globalOffset = chunk.startingIndex
		index.page = i

		data = chunk.data
		if data == nil {
			continue
		}
		for j := chunk.size - 1; j >= 0; j-- {
			index.local = j
			if !yield(index, &data[j]) {
				return
			}
		}
	}
}

func (a *ChunkArray[T]) AllDataParallel(yield func(*T) bool) {
	var chunk *ChunkArrayElement[T]
	var wg = new(sync.WaitGroup)
	var shouldReturn = false

	buffer := a.buffer

	parallelChunks := bits.Len(uint(a.parallelCount)) - 1
	for i := a.bufferSize - 1; i >= 0; i-- {
		chunk = &buffer[i]
		data := chunk.data
		if data == nil {
			continue
		}

		if parallelChunks == 0 {
			for j := chunk.size - 1; j >= 0; j-- {
				if shouldReturn {
					return
				}
				if j >= len(data) {
					continue
				}
				element := &data[j]
				if !yield(element) {
					shouldReturn = true
					return
				}
			}
		} else {
			parallelSubChunks := 1 << (parallelChunks - 1)
			subchunkSize := cap(data) >> (parallelChunks - 1)
			wg.Add(parallelSubChunks)
			for p := 0; p < parallelSubChunks; p++ {
				startIndex := p * subchunkSize
				endIndex := startIndex + subchunkSize
				if endIndex >= chunk.size-1 {
					endIndex = chunk.size
				}
				go func(wg *sync.WaitGroup, stop *bool, data []T, startIndex int, endIndex int, localyield func(*T) bool) {
					defer wg.Done()
					for j := startIndex; j < endIndex; j++ {
						if *stop {
							return
						}
						element := &data[j]
						if j >= len(data) {
							continue
						}
						if !localyield(element) {
							*stop = true
							return
						}
					}
				}(wg, &shouldReturn, data, startIndex, endIndex, yield)
			}
			parallelChunks--
		}
	}
	wg.Wait()
}

func (a *ChunkArray[T]) AllParallel(yield func(ChunkArrayIndex, *T) bool) {
	var chunk *ChunkArrayElement[T]
	var index ChunkArrayIndex
	var wg = new(sync.WaitGroup)
	var shouldReturn = false

	buffer := a.buffer

	parallelChunks := bits.Len(uint(a.parallelCount)) - 1
	for i := a.bufferSize - 1; i >= 0; i-- {
		chunk = &buffer[i]
		data := chunk.data
		if data == nil {
			continue
		}

		index.globalOffset = chunk.startingIndex
		index.page = i

		if parallelChunks == 0 {
			for j := chunk.size - 1; j >= 0; j-- {
				if shouldReturn {
					return
				}
				if j >= len(data) {
					continue
				}
				index.local = j
				if !yield(index, &data[j]) {
					shouldReturn = true
					return
				}
			}
		} else {
			parallelSubChunks := 1 << (parallelChunks - 1)
			subchunkSize := cap(data) >> (parallelChunks - 1)
			wg.Add(parallelSubChunks)
			for p := 0; p < parallelSubChunks; p++ {
				startIndex := p * subchunkSize
				endIndex := startIndex + subchunkSize
				go func(wg *sync.WaitGroup, stop *bool, data []T, index ChunkArrayIndex, startIndex int, endIndex int, localyield func(ChunkArrayIndex, *T) bool) {
					defer wg.Done()
					for j := startIndex; j < endIndex; j++ {
						if *stop {
							return
						}
						if j >= len(data) {
							continue
						}
						index.local = j
						if !localyield(index, &data[j]) {
							*stop = true
							return
						}
					}
				}(wg, &shouldReturn, data, index, startIndex, endIndex, yield)
			}
			parallelChunks--
		}
	}
	wg.Wait()
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
	c.parent.bufferSize++
	c.parent.current = c.next
	c.next.Append(value)

	return
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
	c.parent.bufferSize--
	c.parent.current = c.prev
}

func (c *ChunkArrayElement[T]) Clean() {
	c.data = c.data[:c.size]
}
