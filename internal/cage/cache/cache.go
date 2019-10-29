// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package cache

type DebugEventType string

const (
	MissEvent  DebugEventType = "Miss"
	HitEvent   DebugEventType = "Hit"
	WriteEvent DebugEventType = "Write"
)

type DebugEvent struct {
	Type  DebugEventType
	Key   interface{}
	Value interface{}
}

type DebugLog struct {
	log []DebugEvent
}

func (l *DebugLog) Add(eventType DebugEventType, key, value interface{}) {
	l.log = append(l.log, DebugEvent{Type: eventType, Key: key, Value: value})
}

func (l *DebugLog) All() []DebugEvent {
	all := make([]DebugEvent, len(l.log))
	copy(all, l.log)
	return all
}

func (l *DebugLog) Len() int {
	return len(l.log)
}
