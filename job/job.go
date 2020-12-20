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
  "strconv"
  "io/ioutil"

  "gopkg.in/yaml.v2"
  "github.com/lancs-net/ukbench/log"
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

type Input struct {
  Name string `yaml:"name"`
  Path string `yaml:"path"`
}

type Output struct {
  Name string `yaml:"name"`
  Path string `yaml:"path"`
}

type Run struct {
  Name      string `yaml:"name"`
  Image     string `yaml:"image"`
  Cores     int    `yaml:"cores"`
  Devices []string `yaml:"devices"`
  Cmd       string `yaml:"cmd"`
  Path      string `yaml:"path"`
}

type Job struct {
  Params        []JobParam `yaml:"params"`
  Inputs        []Input    `yaml:"inputs"`
  Outputs       []Output   `yaml:"outputs"`
  Runs          []Run      `yaml:"runs"`
  waitList     *List
}

// RuntimeConfig contains details about the runtime of ukbench
type RuntimeConfig struct {
  Cpus          []int
}

// tasksInFlight represents the maximum tasks which are actively running
// concurrently.  When a tasks is completed, it will leave this list and a
// new task can join.
var tasksInFlight *CoreMap

// NewJob prepares a job yaml file
func NewJob(filePath string, cfg *RuntimeConfig) (*Job, error) {
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

  // Create all tasks for job, iterating over all possible parameter 
  // permutations
  tasks, err := job.tasks()
  if err != nil {
    return nil, err
  }

  // Create a queue of size equal to the number of cores to eventually use
  job.waitList = NewList(len(tasks))

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

    task.Init(&job.Runs)
    job.waitList.Add(task)
  }

  log.Debugf("There are total %d tasks", job.waitList.Len())

  // Prepare a map of cores to hold onto a particular task's run
  tasksInFlight = NewCoreMap(cfg.Cpus)

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
      for i := min; i <= max; math.Pow(float64(step), float64(i)) {
        params = append(params, TaskParam{
          Name:  param.Name,
          Type:  param.Type,
          Value: strconv.Itoa(i),
        })
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

// Start the job
func (j *Job) Start(cfg *RuntimeConfig) error {
  log.Info("Starting job...")

  // TODO: Create a matrix from all the parameters

  return nil
}
