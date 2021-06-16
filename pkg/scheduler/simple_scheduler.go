package scheduler
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
  "time"

  "github.com/lancs-net/ukbench/spec"

  "github.com/lancs-net/ukbench/pkg/runner"

  "github.com/lancs-net/ukbench/internal/log"
  "github.com/lancs-net/ukbench/internal/list"
  "github.com/lancs-net/ukbench/internal/coremap"
)

type SimpleScheduler struct {
  Scheduler
  config         *spec.Runtime
  tasksInFlight  *coremap.CoreMap
  freeCores     []int
  wg              sync.WaitGroup
  waitList       *list.List
  totalTasks      int
  currTaskNum     int
  permutations []*spec.JobPermutation
}

func (ss *SimpleScheduler) Init(config *spec.Runtime, cpus []int, perms []*spec.JobPermutation) error {
  log.Debugf("Initializing simple scheduler...")

  if len(cpus) == 0 {
    return fmt.Errorf("No CPUs provided")
  }

  ss.permutations = perms
  ss.config = config

  // tasksInFlight represents the maximum tasks which are actively running
  // concurrently.  When a tasks is completed, it will leave this list and a
  // new task can join.
  ss.tasksInFlight = coremap.NewCoreMap(cpus)

  // Create a list with all the permutations waiting
  ss.waitList = list.NewList(len(perms))
  
  // Calculate total number of runs per permutation
  ss.totalTasks = 0
  for _, perm := range perms {
    ss.totalTasks += len(*perm.Runs)
  }
  
  ss.currTaskNum = 0

  // Iterate over all the tasks, check if the run is stasifyable, initialize the
  // task and add it to the waiting list.
  for _, perm := range perms {
    for _, r := range *perm.Runs {
      // Check if this particular run has requested more cores than what is
      if r.Cores > len(cpus) {
        return fmt.Errorf(
          "Run has too many cores: %s: %d > %d",
          r.Name,
          r.Cores,
          len(cpus),
        )
      }
    }

    task, err := NewTask(perm,
                         config.WorkDir,
                         config.AllowOverride,
                         config.DryRun)
    if err != nil {
      log.Errorf("Could not initialize task: %s", err)
    } else {
      ss.waitList.Add(task)
    }
  }

  return nil
}

func (ss *SimpleScheduler) Iterator() <- chan func(bridge *runner.Bridge) {
  ch := make(chan func(bridge *runner.Bridge))
  go func() {
    defer close(ch)

    // Continuously iterate over the wait list and the queue of the task to
    // determine whether there is space for the task's run to be scheduled
    // on the available list of cores.
    for i := 0; ss.waitList.Len() > 0; {
      // Continiously updates the number of available cores free so this
      // particular task's run so we can decide whether to schedule it.
      ss.freeCores = ss.tasksInFlight.FreeCores()
      if len(ss.freeCores) == 0 {
        continue
      }

      // Get the next task from the job's queue
      task, err := ss.waitList.Get(i)
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
      } else if len(ss.freeCores) >= nextRun.(spec.Run).Cores {
        // Check if the peaked run is currently active
        ss.tasksInFlight.RLock()
        for _, atr := range ss.tasksInFlight.All() {
          if atr != nil {
            if (atr.(*ActiveTaskRun)).Task.permutation.UUID() == task.(*Task).permutation.UUID() {
              ss.tasksInFlight.RUnlock()
              goto iterator
            }
          }
        }
        ss.tasksInFlight.RUnlock()

        // Select some core IDs for this run based on how many it requires
        var cores []int
        for j := 0; j < nextRun.(spec.Run).Cores; j++ {
          cores = append(cores, ss.freeCores[len(ss.freeCores)-1])
          ss.freeCores = ss.freeCores[:len(ss.freeCores)-1]
        }

        // Initialize the task run
        activeTaskRun, err := NewActiveTaskRun(
          task.(*Task),
          nextRun.(spec.Run),
          cores,
          ss.config.DryRun,
          ss.config.MaxRetries,
        )
        if err != nil {
          log.Errorf("Could not initialize run for this task: %s", err)

          // By cancelling all the subsequent runs, the task will be removed from 
          // scheduler.
          task.(*Task).Cancel()
          goto iterator
        }

        ss.currTaskNum++
        log.Infof("Scheduling task run %s (%d/%d)...",
          activeTaskRun.UUID(),
          ss.currTaskNum,
          ss.totalTasks,
        )

        // Finally, we can dequeue the run since we are about to schedule it
        nextRun, err = task.(*Task).runs.Dequeue()

        // Add the active task to the list of utilised cores
        j := 1
        for len(cores) > 0 {
          coreId := cores[len(cores)-j]
          err := ss.tasksInFlight.Set(coreId, activeTaskRun)
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

        ss.wg.Add(1)

        // Yield a method where we oversee the runtime of this task's run.  By
        // starting this run, it will decide how to consume the cores we have
        // provided to it.
        ch <- func(bridge *runner.Bridge) {
          var returnCode int
          for i := 0; i < activeTaskRun.maxRetries + 1; i++ {
            returnCode, timeElapsed, err := activeTaskRun.Start(bridge)
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
          ss.wg.Done() // We're done here

          // Remove utilized cores from this active task's run
          for _, coreId := range activeTaskRun.CoreIds {
            ss.tasksInFlight.Unset(coreId)
          }
        }
      }

iterator:
      time.Sleep(time.Duration(ss.config.ScheduleGrace) * time.Second)

      // Remove the task if the queue is empty
      if task.(*Task).runs.Len() == 0 {
        ss.waitList.Remove(i)
        i = i - 1
      }

      // Have we reached the end of the list?  Go back to zero otherwise continue.
      if ss.waitList.Len() == i + 1 {
        i = 0
      } else {
        i = i + 1
      }
    }

    // Wait for all controller threads for the task's run to finish
    ss.wg.Wait()
  }()

  return ch
}
