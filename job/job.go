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
  "os"
  "fmt"
  "math"
  "time"
  "sync"
  "path"
  "strconv"
  "io/ioutil"
  "encoding/json"

  "gopkg.in/yaml.v2"
  "github.com/novln/docker-parser"

  "github.com/lancs-net/ukbench/log"
  "github.com/lancs-net/ukbench/run"

  "github.com/lancs-net/ukbench/internal/list"
  "github.com/lancs-net/ukbench/internal/coremap"
)

type JobParam struct {
  Name      string `yaml:"name"`
  Type      string `yaml:"type"`
  Default   string `yaml:"default"`
  Only    []string `yaml:"only"`
  Min       string `yaml:"min"`
  Max       string `yaml:"max"`
  Step      string `yaml:"step"`
  StepMode  string `yaml:"step_mode"`
}

type Job struct {
  Params        []JobParam   `yaml:"params"`
  Inputs        []run.Input  `yaml:"inputs"`
  Outputs       []run.Output `yaml:"outputs"`
  Runs          []run.Run    `yaml:"runs"`
  waitList     *list.List
  scheduleGrace int
  dryRun        bool
  bridge       *run.Bridge
  maxRetries    int
}

// RuntimeConfig contains details about the runtime of ukbench
type RuntimeConfig struct {
  Cpus          []int
  BridgeName      string
  BridgeIface     string
  BridgeSubnet    string
  ScheduleGrace   int
  WorkDir         string
  AllowOverride   bool
  MaxRetries      int
}

// tasksInFlight represents the maximum tasks which are actively running
// concurrently.  When a tasks is completed, it will leave this list and a
// new task can join.
var tasksInFlight *coremap.CoreMap

// NewJob prepares a job yaml file
func NewJob(filePath string, cfg *RuntimeConfig, dryRun bool) (*Job, error) {
  // Check if the path is set
  if len(filePath) == 0 {
    return nil, fmt.Errorf("File path cannot be empty")
  }

  // Check if the file exists
  if _, err := os.Stat(filePath); os.IsNotExist(err) {
    return nil, fmt.Errorf("File does not exist: %s", filePath)
  }

  log.Debugf("Reading job configuration: %s", filePath)

  // Slurp the file contents into memory
  dat, err := ioutil.ReadFile(filePath)
  if err != nil {
    return nil, err
  }

  if len(dat) == 0 {
    return nil, fmt.Errorf("File is empty")
  }

  job := Job{}

  err = yaml.Unmarshal([]byte(dat), &job)
  if err != nil {
    return nil, err
  }

  log.Info("Calculating number of tasks...")

  if len(job.Params) == 0 {
    return nil, fmt.Errorf("You have not set any parameters")
  }

  // Create all tasks for job, iterating over all possible parameter 
  // permutations
  tasks, err := job.tasks()
  if err != nil {
    return nil, err
  }

  // Write a tasks file containing all the permutations
  tasksJson := make(map[string]interface{})
  for _, task := range tasks {
    params := make(map[string]string)
    for _, param := range task.Params {
      params[param.Name] = param.Value
    }
    tasksJson[task.UUID()] = params
  }

  b, err := json.MarshalIndent(tasksJson, "", "\t")
  if err != nil {
    return nil, fmt.Errorf("Could not marshal JSON of tasks: %s", err)
  }

  tasksJsonFile := path.Join(cfg.WorkDir, "results", "tasks.json")
  log.Debugf("Writing tasks file %s...", tasksJsonFile)
  err = ioutil.WriteFile(tasksJsonFile, b, 0644)
  if err != nil {
    return nil, fmt.Errorf("Could not write tasks file: %s", err)
  }

  // Create a list with all the tasks waiting
  job.waitList = list.NewList(len(tasks))

  // Set the schedule grace time
  job.scheduleGrace = cfg.ScheduleGrace

  job.dryRun = dryRun
  job.maxRetries = cfg.MaxRetries

  // Iterate over all the tasks, check if the run is stasifyable, initialize the
  // task and add it to the waiting list.
  for _, task := range tasks {
    for i, run := range job.Runs {
      // Check if this particular run has requested more cores than what is
      if run.Cores > len(cfg.Cpus) {
        return nil, fmt.Errorf(
          "Run has too many cores: %s: %d > %d",
          run.Name,
          run.Cores,
          len(cfg.Cpus),
        )

      // Set the default number of cores to use
      } else if run.Cores == 0 {
        job.Runs[i].Cores = 1
      }
    }

    err := task.Init(cfg.WorkDir, cfg.AllowOverride, &job.Runs, dryRun)
    if err != nil {
      log.Errorf("Could not initialize task: %s", err)
    } else {
      job.waitList.Add(task)
    }
  }

  log.Infof("There are total %d tasks", job.waitList.Len())

  // Prepare a map of cores to hold onto a particular task's run
  tasksInFlight = coremap.NewCoreMap(cfg.Cpus)

  // Set up the bridge
  job.bridge = &run.Bridge{
    Name:      cfg.BridgeName,
    Interface: cfg.BridgeIface,
    Subnet:    cfg.BridgeSubnet,
    CacheDir:  path.Join(cfg.WorkDir, ".cache"),
  }
  err = job.bridge.Init(dryRun)
  if err != nil {
    return nil, fmt.Errorf("Could not create bridge: %s", err)
  }

  return &job, nil
}

// parseParamInt attends to string parameters and its possible permutations
func parseParamStr(param *JobParam) ([]TaskParam, error) {
  var params []TaskParam

  if len(param.Only) > 0 {
    for _, val := range param.Only {
      params = append(params, TaskParam{
        Name:  param.Name,
        Type:  param.Type,
        Value: val,
      })
    }
  } else if len(param.Default) > 0 {
    params = append(params, TaskParam{
      Name:  param.Name,
      Type:  param.Type,
      Value: param.Default,
    })
  }

  return params, nil
}

// parseParamInt attends to integer parameters and its possible permutations
func parseParamInt(param *JobParam) ([]TaskParam, error) {
  var params []TaskParam

  // Parse values in only
  if len(param.Only) > 0 {
    for _, val := range param.Only {
      params = append(params, TaskParam{
        Name:  param.Name,
        Type:  param.Type,
        Value: val,
      })
    }

  // Parse range between min and max
  } else if len(param.Min) > 0 {
    min, err := strconv.Atoi(param.Min)
    if err != nil {
      return nil, err
    }
    
    max, err := strconv.Atoi(param.Max)
    if err != nil {
      return nil, err
    }

    if max < min {
      return nil, fmt.Errorf(
        "Min can't be greater than max for %s: %d < %d", param.Name, min, max,
      )
    }

    // Figure out the step
    step := 1
    if len(param.Step) > 0 {
      step, err = strconv.Atoi(param.Step)
      if err != nil || step == 0 {
        return nil, fmt.Errorf(
          "Invalid step for %s: %s", param.Name, param.Step,
        )
      }
    }

    // Use iterative step
    if len(param.StepMode) == 0 || param.StepMode == "increment" {
      for i := min; i <= max; i += step {
        params = append(params, TaskParam{
          Name:  param.Name,
          Type:  param.Type,
          Value: strconv.Itoa(i),
        })
      }

    // Use exponential step
    } else if param.StepMode == "power" {
      for i, j := min, min; i <= max; j++ {
        params = append(params, TaskParam{
          Name:  param.Name,
          Type:  param.Type,
          Value: strconv.Itoa(i),
        })
        i = int(math.Pow(float64(step), float64(j)))
      }

    // Unknown step mode
    } else {
      return nil, fmt.Errorf(
        "Unknown step mode for param %s: %s", param.Name, param.StepMode,
      )
    }

  } else if len(param.Default) > 0 {
    params = append(params, TaskParam{
      Name:  param.Name,
      Type:  param.Type,
      Value: param.Default,
    })

  } else {
    log.Warnf("Parameter not parsed: %s", param.Name)
  }

  return params, nil
}

// paramPermutations discovers all the possible variants of a particular
// parameter based on its type and options.
func paramPermutations(param *JobParam) ([]TaskParam, error) {
  switch t := param.Type; t {
  case "string":
    return parseParamStr(param)
  case "int":
    return parseParamInt(param)
  case "integer":
    return parseParamInt(param)
  }
  return nil, fmt.Errorf(
    "Unknown parameter type: \"%s\" for %s", param.Type, param.Name,
  )
}

// nextTask recursively iterates across paramters to generate a set of tasks
func (j *Job) nextTask(i int, tasks []*Task, curr []TaskParam) ([]*Task, error) {
  // List all permutations for this parameter
  params, err := paramPermutations(&j.Params[i])
  if err != nil {
    return nil, err
  }

  for _, param := range params {
    if len(curr) > 0 {
      last := curr[len(curr)-1]
      if last.Name == param.Name {
        curr = curr[:len(curr)-1]
      }
    }

    curr = append(curr, param)

    // Break when there are no more parameters to iterate over, thus creating
    // the task.
    if i + 1 == len(j.Params) {
      var p = make([]TaskParam, len(j.Params))
      copy(p, curr)
      task := &Task{
        Inputs:  &j.Inputs,
        Outputs: &j.Outputs,
        Params:   p,
      }
      tasks = append(tasks, task)

    // Otherwise, recursively parse parameters in-order    
    } else {
      nextTasks, err := j.nextTask(i + 1, nil, curr)
      if err != nil {
        return nil, err
      }

      tasks = append(tasks, nextTasks...)
    }
  }

  return tasks, nil
}

// tasks returns a list of all possible tasks based on parameterisation
func (j *Job) tasks() ([]*Task, error) {
  var tasks []*Task

  tasks, err := j.nextTask(0, tasks, nil)
  if err != nil {
    return nil, err
  }

  return tasks, nil
}

// Start the job and all of its tasks
func (j *Job) Start() error {
  var freeCores []int
  var wg sync.WaitGroup

  // Pre-emptively pull all images
  for _, r := range j.Runs {
    ref, err := dockerparser.Parse(r.Image)
    if err != nil {
      return fmt.Errorf("Could not parse image: %s", err)
    }

    log.Infof("Pulling %s...", ref.Remote())

    _, err = run.PullImage(ref.Remote(), j.bridge.CacheDir)
    if err != nil {
      return fmt.Errorf("Could not pull image: %s", err)
    }
  }

  curTaskNum := 0
  totalTasks := j.waitList.Len() * len(j.Runs)

  // Continuously iterate over the wait list and the queue of the task to
  // determine whether there is space for the task's run to be scheduled
  // on the available list of cores.
  for i := 0; j.waitList.Len() > 0; {
    // Continiously updates the number of available cores free so this
    // particular task's run so we can decide whether to schedule it.
    freeCores = tasksInFlight.FreeCores()
    if len(freeCores) == 0 {
      continue
    }

    // Get the next task from the job's queue
    task, err := j.waitList.Get(i)
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
    } else if len(freeCores) >= nextRun.(run.Run).Cores {
      // Check if the peaked run is currently active
      tasksInFlight.RLock()
      for _, atr := range tasksInFlight.All() {
        if atr != nil {
          if (atr.(*ActiveTaskRun)).Task.UUID() == task.(*Task).UUID() {
            tasksInFlight.RUnlock()
            goto iterator
          }
        }
      }
      tasksInFlight.RUnlock()

      // Select some core IDs for this run based on how many it requires
      var cores []int
      for j := 0; j < nextRun.(run.Run).Cores; j++ {
        cores = append(cores, freeCores[len(freeCores)-1])
        freeCores = freeCores[:len(freeCores)-1]
      }

      // Initialize the task run
      activeTaskRun, err := NewActiveTaskRun(
        task.(*Task),
        nextRun.(run.Run),
        cores,
        j.bridge,
        j.dryRun,
        j.maxRetries,
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
    time.Sleep(time.Duration(j.scheduleGrace) * time.Second)

    // Remove the task if the queue is empty
    if task.(*Task).runs.Len() == 0 {
      j.waitList.Remove(i)
      i = i - 1
    }

    // Have we reached the end of the list?  Go back to zero otherwise continue.
    if j.waitList.Len() == i + 1 {
      i = 0
    } else {
      i = i + 1
    }
  }

  wg.Wait() // Wait for all controller threads for the task's run to finish

  return nil
}

// Cleanup provides a way to deschedule all currently active tasks
func (j *Job) Cleanup() {
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
