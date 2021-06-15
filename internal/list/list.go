package list
// SPDX-License-Identifier: BSD-3-Clause
//
// Authors: Alexander Jung <a.jung@lancs.ac.uk>
//
// Copyright (c) 2020, Lancaster University.  All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions
// are met:
//
// 1. Redistributions of source code must retain the above copyright
//    notice, this list of conditions and the following disclaimer.
// 2. Redistributions in binary form must reproduce the above copyright
//    notice, this list of conditions and the following disclaimer in the
//    documentation and/or other materials provided with the distribution.
// 3. Neither the name of the copyright holder nor the names of its
//    contributors may be used to endorse or promote products derived from
//    this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
// ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE
// LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
// CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
// SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
// INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
// CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
// ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
// POSSIBILITY OF SUCH DAMAGE.

import (
  "fmt"
  "sync"
)

// List holds onto a generic out-of-order concurrency-safe array.
type List struct {
  sync.RWMutex
  items []interface{}
}

// NewList creates a generic out-of-order concurrency-safe array.
func NewList(capacity int) *List {
  list := &List{}
  return list
}

// Add to the list
func (l *List) Add(item interface{}) {
  l.RLock()
  l.items = append(l.items, item)
  l.RUnlock()
}

// Remove from the list
func (l *List) Remove(i int) interface{} {
  l.RLock()

  if i > len(l.items) {
    return nil
  }

  ret := l.items[i]  
  l.items = append(l.items[:i], l.items[i+1:]...)

  l.RUnlock()

  return ret
}

// Len returns the length of the list
func (l *List) Len() int {
  l.RLock()
  if l.items == nil {
    l.RUnlock()
    return 0
  }
  ret := len(l.items)
  l.RUnlock()
  return ret
}

// Get an item
func (l *List) Get(i int) (interface{}, error) {
  if i > l.Len() {
    return nil, fmt.Errorf("Could not find element: %d", i)
  }
  
  return l.items[i], nil
}
