package runtime
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
  "os"
  "fmt"
  // "time"
  "path"
  "reflect"
  "io/ioutil"
  "encoding/json"

  "gopkg.in/yaml.v2"
  "github.com/novln/docker-parser"

  "github.com/lancs-net/ukbench/spec"

  "github.com/lancs-net/ukbench/pkg/runner"
  "github.com/lancs-net/ukbench/pkg/scheduler"
  
  "github.com/lancs-net/ukbench/internal/log"
)

// RuntimeActivity contains details about the runtime of ukbench
type RuntimeActivity struct {
  config         *spec.RuntimeSpec
  cpus          []int
  bridge         *runner.Bridge
  job            *spec.Job
  scheduler       scheduler.Scheduler
}

// NewRuntimeActivity
func NewRuntimeActivity(r *spec.RuntimeSpec, cpus []int, jobFile string) (*RuntimeActivity, error) {
  // Check if the path is set
  if len(jobFile) == 0 {
    return nil, fmt.Errorf("File path cannot be empty")
  }

  // Check if the file exists
  if _, err := os.Stat(jobFile); os.IsNotExist(err) {
    return nil, fmt.Errorf("File does not exist: %s", jobFile)
  }

  log.Debugf("Reading job configuration: %s", jobFile)

  // Slurp the file contents into memory
  dat, err := ioutil.ReadFile(jobFile)
  if err != nil {
    return nil, err
  }

  if len(dat) == 0 {
    return nil, fmt.Errorf("File is empty")
  }

  activity := &RuntimeActivity{
    config:  r,
    cpus:    cpus,
    job:    &spec.Job{},
  }

  err = yaml.Unmarshal([]byte(dat), activity.job)
  if err != nil {
    return nil, err
  }

  log.Info("Calculating number of tasks...")

  if len(activity.job.Params) == 0 {
    return nil, fmt.Errorf("You have not set any parameters")
  }

  // Create all permutations for a job, iterating over all possible parameters
  perms, err := activity.job.Permutations()
  if err != nil {
    return nil, err
  }

  // Write a tasks file containing all the permutations
  permsJson := make(map[string]interface{})
  for _, perm := range perms {
    params := make(map[string]string)
    for _, param := range perm.Params {
      params[param.Name] = param.Value
    }
    permsJson[perm.UUID()] = params
  }

  b, err := json.MarshalIndent(permsJson, "", "\t")
  if err != nil {
    return nil, fmt.Errorf("Could not marshal JSON of permutations: %s", err)
  }

  permsJsonFile := path.Join(activity.config.WorkDir, "results", "perms.json")
  log.Debugf("Writing perms file %s...", permsJsonFile)
  err = ioutil.WriteFile(permsJsonFile, b, 0644)
  if err != nil {
    return nil, fmt.Errorf("Could not write perms file: %s", err)
  }

  // log.Infof("There are total %d tasks", activity.waitList.Len())

  // Set up the bridge
  activity.bridge = &runner.Bridge{
    Name:      activity.config.BridgeName,
    Interface: activity.config.HostNetwork,
    Subnet:    activity.config.BridgeSubnet,
    CacheDir:  path.Join(activity.config.WorkDir, ".cache"),
  }
  err = activity.bridge.Init(activity.config.DryRun)
  if err != nil {
    return nil, fmt.Errorf("Could not create bridge: %s", err)
  }

  return activity, nil
}

// Start the job and all of its tasks
func (a *RuntimeActivity) Start() error {
  // Pre-emptively pull all images
  for _, r := range a.job.Runs {
    ref, err := dockerparser.Parse(r.Image)
    if err != nil {
      return fmt.Errorf("Could not parse image: %s", err)
    }

    log.Infof("Pulling %s...", ref.Remote())

    _, err = runner.PullImage(ref.Remote(), a.bridge.CacheDir)
    if err != nil {
      return fmt.Errorf("Could not pull image: %s", err)
    }
  }

  for name, typ := range(scheduler.Schedulers) {
    if a.config.Scheduler == name {
      a.scheduler = reflect.New(typ).Interface().(scheduler.Scheduler)
      break
    }
  }

  if a.scheduler == nil {
    return fmt.Errorf("Could not identify scheduler: %s", a.config.Scheduler)
  }

  perms, err := a.job.Permutations()
  if err != nil {
    return err
  }
  
  err = a.scheduler.Init(a.config, a.cpus, perms)
  if err != nil {
    return fmt.Errorf("Could not initialize scheduler: %s", err)
  }

  iter := a.scheduler.Iterator()

  for next := range iter {
    next(a.bridge)
  }

  return nil
}

// Cleanup provides a way to deschedule all currently active tasks
func (a *RuntimeActivity) Cleanup() {
  a.scheduler.Cleanup()
//   // Iterate through active tasks
//   for _, atr := range tasksInFlight.All() {
//     // Skip cores which do not have a task
//     if atr == nil {
//       continue
//     }

//     err := atr.(*ActiveTaskRun).Runner.Destroy()
//     if err != nil {
//       log.Warnf("Could not destroy runner: %s", err)
//     }
//   }
}
