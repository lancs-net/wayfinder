package cmd
// SPDX-License-Identifier: BSD-3-Clause
//
// Authors: Alexander Jung <a.jung@lancs.ac.uk>
//
// Copyright (c) 2021, Lancaster University.  All rights reserved.
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
  "path"
  "strconv"
  "strings"
  
  "github.com/lancs-net/ukbench/spec"

	"github.com/lancs-net/ukbench/pkg/runtime"
)

var (
  RunCommandDescription = "Run a specific experiment job"
  RunCommandHelp        = RunCommandDescription
)

type RunCommand struct {
  spec.RuntimeSpec
}

func (c *RunCommand) Execute(args []string) error {
  // Determine CPU sets
  cpus, err := parseCpuSets(c.CpuSets)
  if err != nil {
    return fmt.Errorf("Could not parse CPU sets: %s", err)
  }

  // Set the working directory to the current directory if unset
  if c.WorkDir == "" {
    c.WorkDir, err = os.Getwd()
    if err != nil {
      return fmt.Errorf("Could not use current directory as workdir: ", err)
    }
  
  // Check if the set working directory exists, otherwise create it
  } else if _, err := os.Stat(c.WorkDir); os.IsNotExist(err) {
    os.MkdirAll(c.WorkDir, os.ModePerm)
  }

  // Create the caching dir
  cacheDir := path.Join(c.WorkDir, ".cache")
  if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
    os.MkdirAll(cacheDir, os.ModePerm)
  }

  // Create the results dir
  rersultsDir := path.Join(c.WorkDir, "results")
  if _, err := os.Stat(rersultsDir); os.IsNotExist(err) {
    os.MkdirAll(rersultsDir, os.ModePerm)
  }

  activity, err := runtime.NewRuntimeActivity(c, cpus, args[0])
	if err != nil {
		return fmt.Errorf("Could not read configuration: %s", err)
	}

  // Prepare environment
  err = runtime.PrepareEnvironment(cpus, c.DryRun)
  if err != nil {
    return fmt.Errorf("Could not prepare environment: %s", err)
  }

  // Start the job with its various tasks
  err = activity.Start()
  if err != nil {
    return fmt.Errorf("Could not start job: %s", err)
  }

  // We're all done now
  return nil
}

func parseCpuSets(cpuSets string) ([]int, error) {
  var cpus []int
  
  if res := strings.Contains(cpuSets, "-"); res {
    c := strings.Split(cpuSets, "-")
    if len(c) > 2 {
      return cpus, fmt.Errorf("Invalid syntax for CPU sets")
    }

    start, err := strconv.Atoi(c[0])
    if err != nil {
      return cpus, fmt.Errorf("Invalid syntax for CPU sets")
    }

    end, err := strconv.Atoi(c[1])
    if err != nil {
      return cpus, fmt.Errorf("Invalid syntax for CPU sets")
    }
    
    for i := start; i < end; i++ {
      cpus = append(cpus, i)
    }
  }

  if strings.Contains(cpuSets, ",") {
    c := strings.Split(cpuSets, ",")

    for i := range c {
      j, err := strconv.Atoi(c[i])
      if err != nil {
        return cpus, fmt.Errorf("Invalid syntax for CPU sets")
      }

      cpus = append(cpus, j)
    }
  }

  return cpus, nil
}
