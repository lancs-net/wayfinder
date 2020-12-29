package cmd
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
  "strings"
  "strconv"
  "runtime"
  "os/signal"

	"github.com/spf13/cobra"

	"github.com/lancs-net/ukbench/log"
	"github.com/lancs-net/ukbench/job"
)

type RunConfig struct {
  CpuSets       string
  DryRun        bool
  ScheduleGrace int
  WorkDir       string
  AllowOverride bool
  HostNetwork   string
  BridgeName    string
  BridgeSubnet  string
}

var (
  runCmd = &cobra.Command{
    Use: "run [OPTIONS...] [FILE]",
    Short: `Run a specific experiment job`,
    Run: doRunCmd,
    Args: cobra.ExactArgs(1),
    DisableFlagsInUseLine: true,
  }
  runConfig = &RunConfig{}
)

func init() {
  runCmd.PersistentFlags().StringVar(
    &runConfig.CpuSets,
    "cpu-sets",
    fmt.Sprintf("2-%d", runtime.NumCPU()),
    "Specify which CPUs to run experiments on.",
  )
  runCmd.PersistentFlags().BoolVarP(
    &runConfig.DryRun,
    "dry-run",
    "D",
    false,
    "Run without affecting the host or running the jobs.",
  )
  runCmd.PersistentFlags().IntVarP(
    &runConfig.ScheduleGrace,
    "schedule-grace-time",
    "g",
    1,
    "Number of seconds to gracefully wait in the scheduler.",
  )
  runCmd.PersistentFlags().StringVarP(
    &runConfig.WorkDir,
    "workdir",
    "w",
    "",
    "Specify working directory for outputting results, data, file systems, etc.",
  )
  runCmd.PersistentFlags().BoolVarP(
    &runConfig.AllowOverride,
    "allow-override",
    "O",
    false,
    "Override contents in directories (otherwise tasks allowed to fail).",
  )
  runCmd.PersistentFlags().StringVarP(
    &runConfig.HostNetwork,
    "hostnet",
    "n",
    "eth0",
    "",
  )
  runCmd.PersistentFlags().StringVarP(
    &runConfig.BridgeName,
    "bridge",
    "b",
    "ukbench0",
    "",
  )
  runCmd.PersistentFlags().StringVarP(
    &runConfig.BridgeSubnet,
    "subnet",
    "s",
    "172.88.0.1/16",
    "",
  )
}

// doRunCmd 
func doRunCmd(cmd *cobra.Command, args []string) {
  // Determine CPU sets
  cpus, err := parseCpuSets(runConfig.CpuSets)
  if err != nil {
    log.Errorf("Could not parse CPU sets: %s", err)
    os.Exit(1)
  }

  // Set the working directory to the current directory if unset
  if runConfig.WorkDir == "" {
    runConfig.WorkDir, err = os.Getwd()
    if err != nil {
      log.Fatal("Could not use current directory as workdir: ", err)
      os.Exit(1)
    }
  
  // Check if the set working directory exists, otherwise create it
  } else if _, err := os.Stat(runConfig.WorkDir); os.IsNotExist(err) {
    os.MkdirAll(runConfig.WorkDir, os.ModePerm)
  }

  // Create the caching dir
  cacheDir := path.Join(runConfig.WorkDir, ".cache")
  if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
    os.MkdirAll(cacheDir, os.ModePerm)
  }

	j, err := job.NewJob(args[0], &job.RuntimeConfig{
    Cpus:          cpus,
    BridgeName:    runConfig.BridgeName,
    BridgeIface:   runConfig.HostNetwork,
    BridgeSubnet:  runConfig.BridgeSubnet,
    ScheduleGrace: runConfig.ScheduleGrace,
    AllowOverride: runConfig.AllowOverride,
    WorkDir:       runConfig.WorkDir,
    DryRun:        runConfig.DryRun,
  })
	if err != nil {
		log.Fatalf("Could not read configuration: %s", err)
		os.Exit(1)
	}

  setupInterruptHandler()

  // Prepare environment
  err = job.PrepareEnvironment(cpus, runConfig.DryRun)
  if err != nil {
    log.Errorf("Could not prepare environment: %s", err)
    cleanup()
    os.Exit(1)
  }

  // Start the job with its various tasks
  err = j.Start()
  if err != nil {
    log.Errorf("Could not start job: %s", err)
    cleanup()
  }

  // We're all done now
  cleanup()
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

// Create a Ctrl+C trap for reverting machine state
func setupInterruptHandler() {
  c := make(chan os.Signal, 1)
  signal.Notify(c, os.Interrupt)
  go func(){
    <-c
    cleanup()
    os.Exit(1)
  }()
}

// Preserve the host environment
func cleanup() {
  log.Info("Running clean up...")
  job.RevertEnvironment(runConfig.DryRun)
}
