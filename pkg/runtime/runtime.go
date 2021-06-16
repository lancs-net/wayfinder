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
  "time"
  "sync"
  "path"
  "io/ioutil"
  "encoding/json"

  "gopkg.in/yaml.v2"
  "github.com/novln/docker-parser"

  "github.com/lancs-net/ukbench/spec"

  "github.com/lancs-net/ukbench/internal/log"
  "github.com/lancs-net/ukbench/run"

  "github.com/lancs-net/ukbench/internal/list"
  "github.com/lancs-net/ukbench/internal/coremap"
)

// RuntimeConfig contains details about the runtime of ukbench
type RuntimeConfig struct {
  Cpus          []int
  BridgeName      string
  BridgeIface     string
  BridgeSubnet    string
  DryRun          bool
  ScheduleGrace   int
  WorkDir         string
  AllowOverride   bool
  MaxRetries      int
  waitList       *list.List
  bridge         *run.Bridge
  job            *spec.Job
}

// tasksInFlight represents the maximum tasks which are actively running
// concurrently.  When a tasks is completed, it will leave this list and a
// new task can join.
var tasksInFlight *coremap.CoreMap

// NewJob prepares a job yaml file
func (cfg *RuntimeConfig) Prepare(filePath string) error {
  // Check if the path is set
  if len(filePath) == 0 {
    return fmt.Errorf("File path cannot be empty")
  }

  // Check if the file exists
  if _, err := os.Stat(filePath); os.IsNotExist(err) {
    return fmt.Errorf("File does not exist: %s", filePath)
  }

  log.Debugf("Reading job configuration: %s", filePath)

  // Slurp the file contents into memory
  dat, err := ioutil.ReadFile(filePath)
  if err != nil {
    return err
  }

  if len(dat) == 0 {
    return fmt.Errorf("File is empty")
  }

  cfg.job = &spec.Job{}

  err = yaml.Unmarshal([]byte(dat), cfg.job)
  if err != nil {
    return err
  }

  log.Info("Calculating number of tasks...")

  if len(cfg.job.Params) == 0 {
    return fmt.Errorf("You have not set any parameters")
  }

  // Create all permutations for a job, iterating over all possible parameters
  perms, err := cfg.job.Permutations()
  if err != nil {
    return err
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
    return fmt.Errorf("Could not marshal JSON of permutations: %s", err)
  }

  permsJsonFile := path.Join(cfg.WorkDir, "results", "perms.json")
  log.Debugf("Writing perms file %s...", permsJsonFile)
  err = ioutil.WriteFile(permsJsonFile, b, 0644)
  if err != nil {
    return fmt.Errorf("Could not write perms file: %s", err)
  }

  // Create a list with all the perms waiting
  cfg.waitList = list.NewList(len(perms))

  // Iterate over all the tasks, check if the run is stasifyable, initialize the
  // task and add it to the waiting list.
  for _, perm := range perms {
    for i, r := range cfg.job.Runs {
      // Check if this particular run has requested more cores than what is
      if r.Cores > len(cfg.Cpus) {
        return fmt.Errorf(
          "Run has too many cores: %s: %d > %d",
          r.Name,
          r.Cores,
          len(cfg.Cpus),
        )

      // Set the default number of cores to use
      } else if r.Cores == 0 {
        cfg.job.Runs[i].Cores = 1
      }
    }

    task, err := NewTask(perm, cfg.WorkDir, cfg.AllowOverride, &cfg.job.Runs, cfg.DryRun)
    if err != nil {
      log.Errorf("Could not initialize task: %s", err)
    } else {
      cfg.waitList.Add(task)
    }
  }

  log.Infof("There are total %d tasks", cfg.waitList.Len())

  // Prepare a map of cores to hold onto a particular task's run
  tasksInFlight = coremap.NewCoreMap(cfg.Cpus)

  // Set up the bridge
  cfg.bridge = &run.Bridge{
    Name:      cfg.BridgeName,
    Interface: cfg.BridgeIface,
    Subnet:    cfg.BridgeSubnet,
    CacheDir:  path.Join(cfg.WorkDir, ".cache"),
  }
  err = cfg.bridge.Init(cfg.DryRun)
  if err != nil {
    return fmt.Errorf("Could not create bridge: %s", err)
  }

  return nil
}

// Start the job and all of its tasks
func (cfg *RuntimeConfig) Start() error {
  var freeCores []int
  var wg sync.WaitGroup

  // Pre-emptively pull all images
  for _, r := range cfg.job.Runs {
    ref, err := dockerparser.Parse(r.Image)
    if err != nil {
      return fmt.Errorf("Could not parse image: %s", err)
    }

    log.Infof("Pulling %s...", ref.Remote())

    _, err = run.PullImage(ref.Remote(), cfg.bridge.CacheDir)
    if err != nil {
      return fmt.Errorf("Could not pull image: %s", err)
    }
  }

  curTaskNum := 0
  totalTasks := cfg.waitList.Len() * len(cfg.job.Runs)

  // Continuously iterate over the wait list and the queue of the task to
  // determine whether there is space for the task's run to be scheduled
  // on the available list of cores.
  for i := 0; cfg.waitList.Len() > 0; {
    // Continiously updates the number of available cores free so this
    // particular task's run so we can decide whether to schedule it.
    freeCores = tasksInFlight.FreeCores()
    if len(freeCores) == 0 {
      continue
    }

    // Get the next task from the job's queue
    task, err := cfg.waitList.Get(i)
    if err != nil {
      i = 0 // jump back to task 0 in case we overflow
      log.Errorf("Could not get task from wait list: %s", err)
      continue
    }

    // Without removing an in-order run from the queue, peak at it so we can
    // determine whether it is schedulable based on the number of cores which
    // are available.
    nextRun, err := task.(*Task).runs.Peak()
    if err != nil {
      log.Errorf("Could not peak next run for task: %d: %s", i, err)

    // Can we schedule this run?  Use an else if here so we don't ruin the
    // ordering of the iterator `i`
    } else if len(freeCores) >= nextRun.(spec.Run).Cores {
      // Check if the peaked run is currently active
      tasksInFlight.RLock()
      for _, atr := range tasksInFlight.All() {
        if atr != nil {
          if (atr.(*ActiveTaskRun)).Task.permutation.UUID() == task.(*Task).permutation.UUID() {
            tasksInFlight.RUnlock()
            goto iterator
          }
        }
      }
      tasksInFlight.RUnlock()

      // Select some core IDs for this run based on how many it requires
      var cores []int
      for j := 0; j < nextRun.(spec.Run).Cores; j++ {
        cores = append(cores, freeCores[len(freeCores)-1])
        freeCores = freeCores[:len(freeCores)-1]
      }

      // Initialize the task run
      activeTaskRun, err := NewActiveTaskRun(
        task.(*Task),
        nextRun.(spec.Run),
        cores,
        cfg.bridge,
        cfg.DryRun,
        cfg.MaxRetries,
      )
      if err != nil {
        log.Errorf("Could not initialize run for this task: %s", err)

        // By cancelling all the subsequent runs, the task will be removed from 
        // scheduler.
        task.(*Task).Cancel()
        goto iterator
      }

      curTaskNum++
      log.Infof("Scheduling task run %s (%d/%d)...",
        activeTaskRun.UUID(),
        curTaskNum,
        totalTasks,
      )

      // Finally, we can dequeue the run since we are about to schedule it
      nextRun, err = task.(*Task).runs.Dequeue()

      // Add the active task to the list of utilised cores
      j := 1
      for len(cores) > 0 {
        coreId := cores[len(cores)-j]
        err := tasksInFlight.Set(coreId, activeTaskRun)
        if err != nil {
          log.Warnf("Could not schedule task on core ID %d: %s", coreId, err)

          // Use an offset to be able to skip over unavailable cores
          if j >= len(cores) {
            j = 1
          } else {
            j = j + 1
          }
          continue
        }

        // If we are able to use the core, remove it from the list
        cores = cores[:len(cores)-j]
      }

      // Create a thread where we oversee the runtime of this task's run.  By
      // starting this run, it will decide how to consume the cores we have
      // provided to it.
      wg.Add(1) // Update wait group for this thread to complete
      go func() {
        var returnCode int
        for i := 0; i < activeTaskRun.maxRetries + 1; i++ {
          returnCode, timeElapsed, err := activeTaskRun.Start()
          if err != nil {
            log.Errorf(
              "Could not complete run: %s: %s",
              activeTaskRun.UUID(),
              err,
            )
          } else if returnCode != 0 {
            log.Errorf(
              "Could not complete run: %s: exited with return code %d",
              activeTaskRun.UUID(),
              returnCode,
            )
          }

          if timeElapsed > 0 {
            log.Successf("Run %s finished in %s", activeTaskRun.UUID(), timeElapsed)
            goto activeTaskDone
          } else {
            log.Errorf("Run %s finished with errors", activeTaskRun.UUID())
            if i < activeTaskRun.maxRetries + 1 {
              log.Infof("Trying run again (%d/%d)", i + 1, activeTaskRun.maxRetries)
            }
          }
        }

        if err != nil || returnCode != 0 {
          // By cancelling all subsequent runs, the task will be removed from 
          // scheduler.
          task.(*Task).Cancel()
        }

activeTaskDone:
        wg.Done() // We're done here

        // Remove utilized cores from this active task's run
        for _, coreId := range activeTaskRun.CoreIds {
          tasksInFlight.Unset(coreId)
        }
      }()
    }

iterator:
    time.Sleep(time.Duration(cfg.ScheduleGrace) * time.Second)

    // Remove the task if the queue is empty
    if task.(*Task).runs.Len() == 0 {
      cfg.waitList.Remove(i)
      i = i - 1
    }

    // Have we reached the end of the list?  Go back to zero otherwise continue.
    if cfg.waitList.Len() == i + 1 {
      i = 0
    } else {
      i = i + 1
    }
  }

  wg.Wait() // Wait for all controller threads for the task's run to finish

  return nil
}

// Cleanup provides a way to deschedule all currently active tasks
func (cfg *RuntimeConfig) Cleanup() {
  // Iterate through active tasks
  for _, atr := range tasksInFlight.All() {
    // Skip cores which do not have a task
    if atr == nil {
      continue
    }

    err := atr.(*ActiveTaskRun).Runner.Destroy()
    if err != nil {
      log.Warnf("Could not destroy runner: %s", err)
    }
  }
}
