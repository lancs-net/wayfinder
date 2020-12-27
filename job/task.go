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
  "path"
	"crypto/md5"

  "github.com/lancs-net/ukbench/log"
  "github.com/lancs-net/ukbench/run"
)

type TaskParam struct {
  Name  string
  Type  string
  Value string
}

// Task is the specific iterated configuration
type Task struct {
  Params      []TaskParam
  Inputs     *[]Input
  Outputs    *[]Output
  runs         *Queue
  uuid          string
  workDir       string
  cacheDir      string
  AllowOverride bool
}

// Init prepare the task 
func (t *Task) Init(workDir string, allowOverride bool, runs *[]Run) error {
  // Create a queue of runs for this particular task
  t.runs = NewQueue(len(*runs))

  // Set the working directory
  t.workDir = path.Join(workDir, "results", t.UUID())
  t.cacheDir = path.Join(workDir, ".cache")

  // Set additional task configuration
  t.AllowOverride = allowOverride

  // Create a working directory for this task
  if _, err := os.Stat(t.workDir); os.IsNotExist(err) {
    os.MkdirAll(workDir, os.ModePerm)

  // Check if we're allowed to override a non-empty directory
  } else {
    isEmpty, err := IsDirEmpty(t.workDir)
    if err != nil {
      return err
    }
    if !isEmpty && !allowOverride {
      return fmt.Errorf("Task directory not empty: %s", t.workDir)
    }

    os.MkdirAll(workDir, os.ModePerm)
  }

  // Add the runs in-order
  for _, run := range *runs {
    t.runs.Enqueue(run)
  }

  return nil
}

// Cancel the task by removing everything from the queue
func (t *Task) Cancel() {
  log.Warnf("Cancelling task and all subsequent runs")

  // Clear queue of subsequent runs
  t.runs.Clear()
}

func (t *Task) UUID() string {
  if len(t.uuid) == 0 {

    // Calculate the UUID based on a reproducible md5 seed
    md5val := md5.New()
    for _, param := range t.Params {
      io.WriteString(md5val, fmt.Sprintf("%s=%s\n", param.Name, param.Value))
    }

    t.uuid = fmt.Sprintf("%x", md5val.Sum(nil))
  }

  return t.uuid
}

// ActiveTaskRun contains information about a particular task's run.
type ActiveTaskRun struct {
  Task     *Task
  run      *Run
  CoreIds []int // the exact core numbers this task is using
  log      *log.Logger
  workDir   string
}

// NewActiveTaskRun initializes the current task and the run step for the
// the specified cores.
func NewActiveTaskRun(task *Task, run Run, coreIds []int) (*ActiveTaskRun, error) {
  atr := &ActiveTaskRun{
    Task:    task,
    run:    &run,
    CoreIds: coreIds,
  }

  atr.log = &log.Logger{
    LogLevel: log.GetLevel(),
    Prefix:   atr.UUID(),
  }

  return atr, nil
}

// UUID returns the Unique ID for the task and run
func (atr *ActiveTaskRun) UUID() string {
  return fmt.Sprintf("%s-%s", atr.Task.UUID(), atr.run.Name)
}

// Start the task's run
func (atr *ActiveTaskRun) Start() (int, error) {
  atr.log.Infof("Initialising run...")

  // Create the run's working directory
  workDir := path.Join(atr.Task.workDir, atr.run.Name)
  if _, err := os.Stat(workDir); os.IsNotExist(err) {
    atr.log.Debugf("Creating directory: %s", workDir)
    os.MkdirAll(workDir, os.ModePerm)
  }

  config := &run.RunnerConfig{
    Log:           atr.log,
    CacheDir:      atr.Task.cacheDir,
    WorkDir:       workDir,
    AllowOverride: atr.Task.AllowOverride,
    Name:          atr.run.Name,
    Image:         atr.run.Image,
    CpuSets:       []int{},
    Devices:       []string{},
  }
  if atr.run.Path != "" {
    config.Path = atr.run.Path
  } else if atr.run.Cmd != "" {
    config.Cmd = atr.run.Cmd
  } else {
    return 1, fmt.Errorf("Run did not specify path or cmd: %s", atr.run.Name)
  }
  runner, err := run.NewRunner(config)
  if err != nil {
    return 1, err
  }

  exitCode, err := runner.Run()
  if err != nil {
    runner.Destroy()
    return 1, fmt.Errorf("Could not start runner: %s", err)
  }

  return exitCode, nil
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
