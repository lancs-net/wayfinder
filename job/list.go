package job
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
  "sync"
)

// CoreMap holds onto to a reference to the particular task which is currently
// running on the core number defined as the index.
type CoreMap struct {
  sync.RWMutex
  cores map[int]*ActiveTaskRun
}

// CoreMap creates a fixed-length map of cores with their ID as index.
func NewCoreMap(cores []int) *CoreMap {
  coreMap := &CoreMap{
    cores: make(map[int]*ActiveTaskRun, len(cores)),
  }

  // Add the core ID as index to the map
  for i := 0; i < len(cores); i++ {
    coreMap.cores[cores[i]] = nil
  }

  return coreMap
}

// Retrieve a list of core numbers whch are free
func (cm *CoreMap) FreeCores() []int {
  var free []int
  cm.RLock()
  for i, _ := range cm.cores {
    if cm.cores[i] == nil {
      free = append(free, i)
    }
  }
  cm.RUnlock()
  return free
}

// Set updates the core ID with the task which is actively using it
func (cm *CoreMap) Set(coreId int, atr *ActiveTaskRun) error {
  cm.Lock()
  if cm.cores[coreId] != nil {
    cm.Unlock()
    return fmt.Errorf("Core already in use by: %#v", cm.cores[coreId])
  }

  log.Debugf("Reserving coreId=%d", coreId)
  cm.cores[coreId] = atr
  cm.Unlock()
  return nil
}

// Get retrieves the ActiveTaskRun at the coreId
func (cm *CoreMap) Get(coreId int) *ActiveTaskRun {
  var atr *ActiveTaskRun
  
  cm.RLock()
  if cm.cores[coreId] != nil {
    atr = cm.cores[coreId]
  }
  cm.RUnlock()
  
  return atr
}

// Unset updates the core ID to be free
func (cm *CoreMap) Unset(coreId int) {
  cm.Lock()
  log.Debugf("Releasing coreId=%d", coreId)
  cm.cores[coreId] = nil
  cm.Unlock()
}
