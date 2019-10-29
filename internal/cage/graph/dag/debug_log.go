// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package dag

type DebugEventType string

const (
	AddEvent     DebugEventType = "Add"
	ConnectEvent DebugEventType = "Connect"
)

type DebugEvent struct {
	Type   DebugEventType
	Detail interface{}
}

type DebugLog struct {
	log []DebugEvent
}

func (l *DebugLog) Add(eventType DebugEventType, detail interface{}) {
	l.log = append(l.log, DebugEvent{Type: eventType, Detail: detail})
}

func (l *DebugLog) All() []DebugEvent {
	all := make([]DebugEvent, len(l.log))
	copy(all, l.log)
	return all
}

func (l *DebugLog) Len() int {
	return len(l.log)
}
