// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package dag

import (
	"fmt"
	"sync"

	"github.com/hashicorp/terraform/dag"
	"github.com/pkg/errors"

	cage_cache "github.com/codeactual/transplant/internal/cage/cache"
)

type hasPathResultMap map[interface{}]bool

// Graph wraps dag.AcyclicGraph in order to provide a restricted, smaller API with modified behaviors
// such as additional error checks and validation.
type Graph struct {
	ag dag.AcyclicGraph

	// hasPathCache allows HasPath* methods to end early if prior walks to the same destination
	// have stepped on the same vertex.
	//
	// Its keys are destination vertices from past HasPath* searches. Its values map vertices
	// with known search results. So reading from it is a process of first answering "where is
	// the HasPath* walk trying to reach?" and then "where is the walk currently at?".
	//
	// It is invalidated as a whole after any graph change.
	hasPathCache map[interface{}]hasPathResultMap

	debugLog      DebugLog
	cacheDebugLog cage_cache.DebugLog
}

func NewGraph() Graph {
	g := Graph{}
	g.initCache()
	return g
}

func (g *Graph) Add(vertex interface{}) {
	g.ag.Add(vertex)
	g.initCache()
	g.debugLog.Add(AddEvent, vertex)
}

func (g *Graph) String() string {
	return g.ag.StringWithNodeTypes()
}

func (g *Graph) HasVertex(v interface{}) bool {
	return g.ag.HasVertex(v)
}

func (g *Graph) HasEdge(start, end interface{}) bool {
	return g.ag.HasEdge(dag.BasicEdge(start, end))
}

func (g *Graph) VerticesFrom(origin interface{}) (vertices []interface{}) {
	from := g.ag.EdgesFrom(origin)
	for _, edge := range from {
		vertices = append(vertices, edge.Target())
	}
	return vertices
}

func (g *Graph) Connect(start, end interface{}) error {
	if !g.HasVertex(start) {
		return errors.Errorf("failed to connect vertices, start vertex [%+v] not in graph", start)
	}
	if !g.HasVertex(end) {
		return errors.Errorf("failed to connect vertices, end vertex [%+v] not in graph", end)
	}
	g.ag.Connect(dag.BasicEdge(start, end))
	g.debugLog.Add(ConnectEvent, []interface{}{start, end})
	return nil
}

func (g *Graph) HasPathDepthFirst(start, end interface{}) (bool, error) {
	if err := g.ag.Validate(); err != nil {
		return false, errors.Wrapf(err, "failed to determine if path exists to [%+v], invalid graph", start)
	}

	// dag walkers parallelize when possible. Although we're only providing one root at a time,
	// we'll lock anyway in case they still (or eventually) parallelize in this scenario.
	var lock sync.Mutex

	var hasPath bool
	var steps []dag.Vertex
	var done bool

	err := g.ag.DepthFirstWalk([]dag.Vertex{start.(dag.Vertex)}, func(v dag.Vertex, depth int) error {
		lock.Lock()
		defer lock.Unlock()

		if done {
			return nil
		}

		hit, cachedHasPath := g.readHasPathCache(v, end)
		if hit {
			hasPath = cachedHasPath
			done = true
			return nil
		}

		steps = append(steps, v)

		if v == end {
			hasPath = true
			done = true
		}

		return nil
	})

	if err != nil {
		return false, errors.Wrapf(err, "failed to determine if path exists to [%+v]", start)
	}

	g.writeHasPathCache(steps, end, hasPath)

	return hasPath, nil
}

func (g *Graph) Copy() (c Graph, err error) {
	for _, v := range g.ag.Vertices() {
		c.Add(v)
	}
	for _, e := range g.ag.Edges() {
		src := e.Source()
		target := e.Target()
		if err := c.Connect(src, target); err != nil {
			return Graph{}, errors.Wrapf(err, "failed to connect source [%+v] to target [%+v]\n", src, target)
		}
	}
	return c, nil
}

func (g *Graph) DebugLogEvents() []DebugEvent {
	return g.debugLog.All()
}

func (g *Graph) readHasPathCache(step dag.Vertex, end dag.Vertex) (hit, hasPath bool) {
	if _, endOk := g.hasPathCache[end]; endOk {
		if _, stepOk := g.hasPathCache[end][step]; stepOk {
			g.cacheDebugLog.Add(cage_cache.HitEvent, fmt.Sprintf("end=%+v step=%+v", end, step), g.hasPathCache[end][step])
			return true, g.hasPathCache[end][step]
		}
	}
	g.cacheDebugLog.Add(cage_cache.MissEvent, fmt.Sprintf("end=%+v step=%+v", end, step), nil)
	return false, false
}

func (g *Graph) writeHasPathCache(steps []dag.Vertex, end dag.Vertex, hasPath bool) {
	if _, ok := g.hasPathCache[end]; !ok {
		g.hasPathCache[end] = make(hasPathResultMap)
	}
	for _, step := range steps {
		g.hasPathCache[end][step] = hasPath
		g.cacheDebugLog.Add(cage_cache.WriteEvent, fmt.Sprintf("end=%+v step=%+v", end, step), hasPath)
	}
}

func (g *Graph) initCache() {
	g.cacheDebugLog = cage_cache.DebugLog{}
	g.hasPathCache = make(map[interface{}]hasPathResultMap)
}
