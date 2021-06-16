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
  "io"
  "os"
  "fmt"
  "time"
  "path"
  "strings"

  "github.com/lancs-net/ukbench/internal/log"
  "github.com/lancs-net/ukbench/internal/queue"

  "github.com/lancs-net/ukbench/run"
  "github.com/lancs-net/ukbench/spec"
)

// Task is the specific iterated configuration
type Task struct {
  permutation  *spec.JobPermutation
  runs         *queue.Queue
  resultsDir    string
  cacheDir      string
  AllowOverride bool
}

// Init prepare the task 
func NewTask(perm *spec.JobPermutation, workDir string, allowOverride bool, runs *[]spec.Run, dryRun bool) (*Task, error) {
  t := &Task{
    permutation: perm,
  }

  // Create a queue of runs for this particular task
  t.runs = queue.NewQueue(len(*runs))

  // Set the working directory
  t.resultsDir = path.Join(workDir, "results", t.permutation.UUID())
  t.cacheDir = path.Join(workDir, ".cache")

  // Set additional task configuration
  t.AllowOverride = allowOverride

  // Create a results directory for this task
  if _, err := os.Stat(t.resultsDir); os.IsNotExist(err) {
    if !dryRun {
      os.MkdirAll(t.resultsDir, os.ModePerm)
    }

  // Check if we're allowed to override a non-empty directory
  } else {
    isEmpty, err := IsDirEmpty(t.resultsDir)
    if err != nil {
      return nil, err
    }
    if !isEmpty && !allowOverride {
      return nil, fmt.Errorf("Task directory not empty: %s", t.resultsDir)
    }

    if !dryRun {
      os.MkdirAll(workDir, os.ModePerm)
    }
  }

  // Add the runs in-order
  for _, run := range *runs {
    t.runs.Enqueue(run)
  }

  return t, nil
}

// Cancel the task by removing everything from the queue
func (t *Task) Cancel() {
  log.Warnf("Cancelling task and all subsequent runs")

  // Clear queue of subsequent runs
  t.runs.Clear()
}

// ActiveTaskRun contains information about a particular task's run.
type ActiveTaskRun struct {
  Task       *Task
  Runner     *run.Runner
  run        *spec.Run
  CoreIds   []int // the exact core numbers this task is using
  log        *log.Logger
  workDir     string
  dryRun      bool
  bridge     *run.Bridge
  maxRetries  int
}

// NewActiveTaskRun initializes the current task and the run step for the
// the specified cores.
func NewActiveTaskRun(task *Task, run spec.Run, coreIds []int, bridge *run.Bridge, dryRun bool, maxRetries int) (*ActiveTaskRun, error) {
  atr := &ActiveTaskRun{
    Task:       task,
    run:       &run,
    CoreIds:    coreIds,
    maxRetries: maxRetries,
  }

  atr.log = &log.Logger{
    LogLevel: log.GetLevel(),
    Prefix:   atr.UUID(),
  }

  atr.bridge = bridge

  return atr, nil
}

// UUID returns the Unique ID for the task and run
func (atr *ActiveTaskRun) UUID() string {
  return fmt.Sprintf("%s-%s", atr.Task.permutation.UUID(), atr.run.Name)
}

// Start the task's run
func (atr *ActiveTaskRun) Start() (int, time.Duration, error) {
  var env []string
  var err error

  for _, param := range atr.Task.permutation.Params {
    env = append(env, fmt.Sprintf("%s=%s", param.Name, param.Value))
  }

  env = append(env, fmt.Sprintf("UKBENCH_TOTAL_CORES=%d", len(atr.CoreIds)))
  env = append(env, fmt.Sprintf("UKBENCH_CORES=%s", strings.Trim(
    strings.Join(strings.Fields(fmt.Sprint(atr.CoreIds)), " "), "[]",
  )))
  for i, coreId := range atr.CoreIds {
    env = append(env, fmt.Sprintf("UKBENCH_CORE_ID%d=%d", i, coreId))
  }

  config := &run.RunnerConfig{
    Log:           atr.log,
    CacheDir:      atr.Task.cacheDir,
    ResultsDir:    atr.Task.resultsDir,
    AllowOverride: atr.Task.AllowOverride,
    Name:          atr.run.Name,
    Image:         atr.run.Image,
    CoreIds:       atr.CoreIds,
    Devices:       atr.run.Devices,
    Inputs:        atr.Task.permutation.Inputs,
    Outputs:       atr.Task.permutation.Outputs,
    Env:           env,
    Capabilities:  atr.run.Capabilities,
  }
  if atr.run.Path != "" {
    config.Path = atr.run.Path
  } else if atr.run.Cmd != "" {
    config.Cmd = atr.run.Cmd
  } else {
    return 1, -1, fmt.Errorf("Run did not specify path or cmd: %s", atr.run.Name)
  }

  atr.Runner, err = run.NewRunner(config, atr.bridge, atr.dryRun)
  if err != nil {
    return 1, -1, err
  }

  atr.log.Infof("Starting run...")
  exitCode, timeElapsed, err := atr.Runner.Run()
  atr.Runner.Destroy()
  if err != nil {
    return 1, -1, fmt.Errorf("Could not start runner: %s", err)
  }

  return exitCode, timeElapsed, nil
}

// IsDirEmpty is a method used to determine whether a directory is empty
func IsDirEmpty(path string) (bool, error) {
  f, err := os.Open(path)
  if err != nil {
    return false, err
  }

  defer f.Close()

  _, err = f.Readdirnames(1) // Or f.Readdir(1)
  if err == io.EOF {
    return true, nil
  }

  // Either not empty or error, suits both cases
  return false, err
}
