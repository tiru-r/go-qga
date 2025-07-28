// Copyright 2025 PREVOST Corentin
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package common

import (
	"net"
	"time"
)

// Simple time utilities - no over-engineered managers needed

// SetConnectionTimeout sets deadline on connection - simple and direct
func SetConnectionTimeout(conn net.Conn, timeout time.Duration) error {
	return conn.SetDeadline(time.Now().Add(timeout))
}

// SetDefaultTimeout sets a standard timeout on connection
func SetDefaultTimeout(conn net.Conn) error {
	return SetConnectionTimeout(conn, DefaultTimeout)
}

// TimeManager provides simple timeout management
type TimeManager struct{}

// SetReadDeadline sets read deadline with specified timeout
func (t *TimeManager) SetReadDeadline(conn net.Conn, timeout time.Duration) error {
	return conn.SetReadDeadline(time.Now().Add(timeout))
}

// SetReadDeadlineDefault sets default read deadline
func (t *TimeManager) SetReadDeadlineDefault(conn net.Conn) error {
	return conn.SetReadDeadline(time.Now().Add(DefaultTimeout))
}

// SetWriteDeadlineDefault sets default write deadline
func (t *TimeManager) SetWriteDeadlineDefault(conn net.Conn) error {
	return conn.SetWriteDeadline(time.Now().Add(DefaultTimeout))
}

// SetDeadline sets general deadline
func (t *TimeManager) SetDeadline(conn net.Conn, timeout time.Duration) error {
	return conn.SetDeadline(time.Now().Add(timeout))
}

// ClearDeadlines clears all deadlines
func (t *TimeManager) ClearDeadlines(conn net.Conn) error {
	return conn.SetDeadline(time.Time{})
}

// NowPlus returns time.Now() + duration
func (t *TimeManager) NowPlus(d time.Duration) time.Time {
	return time.Now().Add(d)
}

// GlobalTimeManager provides global access to time utilities
var GlobalTimeManager = &TimeManager{}