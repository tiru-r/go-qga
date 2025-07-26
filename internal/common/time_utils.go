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

// TimeManager provides centralized time management utilities
type TimeManager struct {
	defaultTimeout time.Duration
	fastTimeout    time.Duration
	slowTimeout    time.Duration
}

// NewTimeManager creates a new time manager
func NewTimeManager() *TimeManager {
	return &TimeManager{
		defaultTimeout: DefaultTimeout,
		fastTimeout:    FastTimeout,
		slowTimeout:    SlowTimeout,
	}
}

// GetDefaultTimeout returns the default timeout
func (tm *TimeManager) GetDefaultTimeout() time.Duration {
	return tm.defaultTimeout
}

// GetFastTimeout returns the fast timeout
func (tm *TimeManager) GetFastTimeout() time.Duration {
	return tm.fastTimeout
}

// GetSlowTimeout returns the slow timeout
func (tm *TimeManager) GetSlowTimeout() time.Duration {
	return tm.slowTimeout
}

// NowPlusDefault returns current time plus default timeout
func (tm *TimeManager) NowPlusDefault() time.Time {
	return time.Now().Add(tm.defaultTimeout)
}

// NowPlusFast returns current time plus fast timeout
func (tm *TimeManager) NowPlusFast() time.Time {
	return time.Now().Add(tm.fastTimeout)
}

// NowPlusSlow returns current time plus slow timeout
func (tm *TimeManager) NowPlusSlow() time.Time {
	return time.Now().Add(tm.slowTimeout)
}

// NowPlus returns current time plus specified duration
func (tm *TimeManager) NowPlus(duration time.Duration) time.Time {
	return time.Now().Add(duration)
}

// SetReadDeadlineDefault sets read deadline with default timeout
func (tm *TimeManager) SetReadDeadlineDefault(conn net.Conn) error {
	return conn.SetReadDeadline(tm.NowPlusDefault())
}

// SetWriteDeadlineDefault sets write deadline with default timeout
func (tm *TimeManager) SetWriteDeadlineDefault(conn net.Conn) error {
	return conn.SetWriteDeadline(tm.NowPlusDefault())
}

// SetDeadlineDefault sets both read and write deadlines with default timeout
func (tm *TimeManager) SetDeadlineDefault(conn net.Conn) error {
	return conn.SetDeadline(tm.NowPlusDefault())
}

// SetReadDeadlineFast sets read deadline with fast timeout
func (tm *TimeManager) SetReadDeadlineFast(conn net.Conn) error {
	return conn.SetReadDeadline(tm.NowPlusFast())
}

// SetWriteDeadlineFast sets write deadline with fast timeout
func (tm *TimeManager) SetWriteDeadlineFast(conn net.Conn) error {
	return conn.SetWriteDeadline(tm.NowPlusFast())
}

// SetDeadlineFast sets both deadlines with fast timeout
func (tm *TimeManager) SetDeadlineFast(conn net.Conn) error {
	return conn.SetDeadline(tm.NowPlusFast())
}

// SetReadDeadlineSlow sets read deadline with slow timeout
func (tm *TimeManager) SetReadDeadlineSlow(conn net.Conn) error {
	return conn.SetReadDeadline(tm.NowPlusSlow())
}

// SetWriteDeadlineSlow sets write deadline with slow timeout
func (tm *TimeManager) SetWriteDeadlineSlow(conn net.Conn) error {
	return conn.SetWriteDeadline(tm.NowPlusSlow())
}

// SetDeadlineSlow sets both deadlines with slow timeout
func (tm *TimeManager) SetDeadlineSlow(conn net.Conn) error {
	return conn.SetDeadline(tm.NowPlusSlow())
}

// SetReadDeadline sets read deadline with custom timeout
func (tm *TimeManager) SetReadDeadline(conn net.Conn, timeout time.Duration) error {
	return conn.SetReadDeadline(tm.NowPlus(timeout))
}

// SetWriteDeadline sets write deadline with custom timeout
func (tm *TimeManager) SetWriteDeadline(conn net.Conn, timeout time.Duration) error {
	return conn.SetWriteDeadline(tm.NowPlus(timeout))
}

// SetDeadline sets both deadlines with custom timeout
func (tm *TimeManager) SetDeadline(conn net.Conn, timeout time.Duration) error {
	return conn.SetDeadline(tm.NowPlus(timeout))
}

// ClearDeadlines clears all deadlines on a connection
func (tm *TimeManager) ClearDeadlines(conn net.Conn) error {
	zeroTime := time.Time{}
	if err := conn.SetReadDeadline(zeroTime); err != nil {
		return err
	}
	return conn.SetWriteDeadline(zeroTime)
}

// SleepDefault sleeps for default timeout duration
func (tm *TimeManager) SleepDefault() {
	time.Sleep(tm.defaultTimeout)
}

// SleepFast sleeps for fast timeout duration
func (tm *TimeManager) SleepFast() {
	time.Sleep(tm.fastTimeout)
}

// SleepSlow sleeps for slow timeout duration
func (tm *TimeManager) SleepSlow() {
	time.Sleep(tm.slowTimeout)
}

// Global time manager instance
var GlobalTimeManager = NewTimeManager()

// Convenience functions using global time manager

// NowPlusDefault returns current time plus default timeout
func NowPlusDefault() time.Time {
	return GlobalTimeManager.NowPlusDefault()
}

// NowPlusFast returns current time plus fast timeout
func NowPlusFast() time.Time {
	return GlobalTimeManager.NowPlusFast()
}

// NowPlusSlow returns current time plus slow timeout
func NowPlusSlow() time.Time {
	return GlobalTimeManager.NowPlusSlow()
}

// SetDeadlineDefault sets both deadlines with default timeout
func SetDeadlineDefault(conn net.Conn) error {
	return GlobalTimeManager.SetDeadlineDefault(conn)
}

// SetDeadlineFast sets both deadlines with fast timeout
func SetDeadlineFast(conn net.Conn) error {
	return GlobalTimeManager.SetDeadlineFast(conn)
}

// SetDeadlineSlow sets both deadlines with slow timeout
func SetDeadlineSlow(conn net.Conn) error {
	return GlobalTimeManager.SetDeadlineSlow(conn)
}