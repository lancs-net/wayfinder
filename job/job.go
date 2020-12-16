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
  "io/ioutil"

  "gopkg.in/yaml.v2"
  log "github.com/sirupsen/logrus"
)

type JobParam struct {
  Name      string `yaml:"name"`
  Type      string `yaml:"type"`
  Default   string `yaml:"default"`
  Only    []string `yaml:"only"`
  Min       string `yaml:"min"`
  Max       string `yaml:"max"`
  Step      int    `yaml:"step"`
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
  Path string `yaml:"path"`
  Cmd  string `yaml:"cmd"`
}

type Job struct {
  Params  []JobParam `yaml:"params"`
  Inputs  []Input    `yaml:"inputs"`
  Outputs []Output   `yaml:"outputs"`
  Run       Run      `yaml:"run"`
}

// NewJob prepares a job yaml file
func NewJob(filePath string) (*Job, error) {
  // Check if the path is set
  if len(filePath) == 0 {
    return nil, fmt.Errorf("File path cannot be empty")
  }

  // Check if the file exists
  if _, err := os.Stat(filePath); os.IsNotExist(err) {
    return nil, fmt.Errorf("File does not exist: %s", filePath)
  }

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

  log.Debugf("Read in job configuration: %s", filePath)

  return &job, nil
}

// RuntimeConfig contains details about the runtime of ukbench
type RuntimeConfig struct {
  CpuSets string
}

// Start the job
func (j *Job) Start(cfg *RuntimeConfig) error {
  log.Info("Starting job...")

  return nil
}
