/*
This Source Code Form is subject to the terms of the Mozilla
Public License, v. 2.0. If a copy of the MPL was not distributed
with this file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

package ecs

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestComponentManager(t *testing.T) {
	sp := CreateComponentManager[string](1)

	sp.Create(10, "foo")
	sp.Create(15, "bar")
	sp.Create(25, "baz")
	sp.Create(23, "qux")
	sp.Create(34, "quux")
	sp.Create(56, "corge")
	sp.Create(78, "grault")

	sp.Remove(23)
	sp.Clean()

	sp.Create(1, "garply")
	sp.Create(44, "waldo")
	sp.Create(51, "fred")

	sp.Remove(10)
	sp.Clean()

	sp.Create(0, "plugh")
	sp.Create(88, "xyzzy")
	sp.Create(91, "thud")

	sp.Remove(88)
	sp.Clean()

	require.Equal(t, "fred", *sp.Get(51))
	require.Equal(t, "baz", *sp.Get(25))
	require.Equal(t, "corge", *sp.Get(56))
	require.Equal(t, "thud", *sp.Get(91))

	sp.Remove(91)
	sp.Clean()

	require.Nil(t, sp.Get(10))
	require.Nil(t, sp.Get(23))
	require.Nil(t, sp.Get(88))
	require.Nil(t, sp.Get(91))
}

func TestComponentManagerMass(t *testing.T) {
	sp := CreateComponentManager[string](1)

	const count = 1_000_000
	for n := 0; n < count; n += 2 {
		sp.Create(EntityID(n), strconv.Itoa(n))
		sp.Create(EntityID(n+1), strconv.Itoa(n+1))
	}

	// test
	for n := 0; n < count; n++ {
		val := sp.Get(EntityID(n))
		if expected := strconv.Itoa(n); *val != expected {
			t.Errorf("expected %q, got %q", expected, *val)
		}
	}

	last := sp.Get(count - 1)
	*last = "last"

	// delete all thirds
	for n := 3; n < count; n += 3 {
		if n == count-1 {
			continue
		}
		sp.Remove(EntityID(n))
	}
	sp.Clean()

	// replace all fifths
	for n := 5; n < count; n += 5 {
		if n == count-1 {
			continue
		}
		sp.Create(EntityID(n), "foo"+strconv.Itoa(n))
	}

	// test again
	for n := 0; n < count; n++ {
		val := sp.Get(EntityID(n))
		switch {
		case n == count-1:
			if expected := "last"; *val != expected {
				t.Errorf("expected %q, got %q", expected, *val)
			}
		case n > 0 && n%5 == 0:
			if expected := "foo" + strconv.Itoa(n); *val != expected {
				t.Errorf("expected %q, got %q", expected, *val)
			}
		case n > 0 && n%3 == 0:
			if val != nil {
				t.Errorf("expected nil, got %q", *val)
			}
		default:
			if expected := strconv.Itoa(n); *val != expected {
				t.Errorf("expected %q, got %q", expected, *val)
			}
		}
	}
}

func TestComponentManagerEach(t *testing.T) {
	t.Run("100_000", func(t *testing.T) {
		type some struct {
			X, Y, Z int
		}
		sp := CreateComponentManager[some](1)
		for i := 0; i < 100_000; i++ {
			sp.Create(EntityID(i), some{})
		}

		started := time.Now()
		var y int
		for _, val := range sp.All {
			val.X = 1
			val.Y = y
			val.Z = -y
			y++
		}
		t.Logf("done 100k with %v", time.Since(started))

		// check
		var z int
		for i := 100_000 - 1; i >= 0; i-- {
			v := sp.Get(EntityID(i))
			require.NotNil(t, v)
			require.Equal(t, 1, v.X)
			require.Equal(t, z, v.Y)
			require.Equal(t, -z, v.Z)
			z++
		}
	})
	t.Run("1_000_000", func(t *testing.T) {
		type some struct {
			X, Y, Z int
		}
		sp := CreateComponentManager[some](1)
		for i := 0; i < 1_000_000; i++ {
			sp.Create(EntityID(i), some{})
		}

		started := time.Now()
		var y int
		for _, val := range sp.All {
			val.X = 1
			val.Y = y
			val.Z = -y
			y++
		}
		t.Logf("done 1m with %v", time.Since(started))

		// check
		var z int
		for i := 1_000_000 - 1; i >= 0; i-- {
			v := sp.Get(EntityID(i))
			require.NotNil(t, v)
			require.Equal(t, 1, v.X)
			require.Equal(t, z, v.Y)
			require.Equal(t, -z, v.Z)
			z++
		}
	})
	t.Run("16_000_000", func(t *testing.T) {
		type some struct {
			X, Y, Z int
		}
		sp := CreateComponentManager[some](1)
		for i := 0; i < 16_000_000; i++ {
			sp.Create(EntityID(i), some{})
		}

		started := time.Now()
		var y int
		for _, val := range sp.All {
			val.X = 1
			val.Y = y
			val.Z = -y
			y++
		}
		t.Logf("done 16m with %v", time.Since(started))
	})
}

func BenchmarkComponentManager(b *testing.B) {
	b.Run("insert", func(b *testing.B) {
		sp := CreateComponentManager[string](1)
		for i := 0; i < 1000000; i++ {
			sp.Create(EntityID(i), fmt.Sprintf("inited-%d", i))
		}

		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if i >= 1000000 {
				sp.Create(EntityID(i), fmt.Sprintf("inserted-%d", i))
				continue
			}
			v := sp.Get(EntityID(i))
			*v = fmt.Sprintf("inserted-%d", i)
		}
	})

	b.Run("delete", func(b *testing.B) {
		sp := CreateComponentManager[string](1)
		for i := 0; i < b.N; i++ {
			sp.Create(EntityID(i), strconv.Itoa(i))
		}

		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			sp.Remove(EntityID(i))
		}
		sp.Clean()
	})
}

func BenchmarkComponentManagerDelete(b *testing.B) {
	sp := CreateComponentManager[string](1)
	for i := 0; i < b.N; i++ {
		sp.Create(EntityID(i), fmt.Sprintf("inited-%d", i))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sp.Remove(EntityID(i))
	}
	sp.Clean()
}

func BenchmarkComponentManagerEach(b *testing.B) {
	b.Run("by_count", func(b *testing.B) {
		sp := CreateComponentManager[uint64](1)
		for i := 0; i < b.N; i++ {
			sp.Create(EntityID(i), uint64(i))
		}
		b.ReportAllocs()
		b.ResetTimer()
		for _, val := range sp.All {
			*val = *val + 12
		}

	})
	b.Run("static_size_1_000_000", func(b *testing.B) {
		sp := CreateComponentManager[uint64](1)
		for i := 0; i < 1_000_000; i++ {
			sp.Create(EntityID(i), uint64(i))
		}
		b.ReportAllocs()
		b.ResetTimer()
		var left = b.N
		for left > 0 {
			for _, val := range sp.All {
				left--
				*val = *val + 12
				if left > 0 {
					continue
				}
			}
		}
	})
}

func BenchmarkComponentManagerEach18m(b *testing.B) {
	sp := CreateComponentManager[uint64](1)
	for i := 0; i < 18_000_000; i++ {
		sp.Create(EntityID(i), uint64(i))
	}
	b.ResetTimer()
	b.Run("static_size", func(b *testing.B) {
		b.ReportAllocs()
		var left = b.N
		for left > 0 {
			for _, val := range sp.All {
				left--
				*val = *val + 12
				if left > 0 {
					continue
				}
			}
		}
	})
}
