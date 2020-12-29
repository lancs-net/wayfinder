package run
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

  "github.com/lancs-net/ukbench/log"
)

type RunnerType int

const (
  UNKNOWN RunnerType = iota
  EMPTY
  RUNC
)

type Input struct {
  Name string `yaml:"name"`
  Path string `yaml:"path"`
}

type Output struct {
  Name string `yaml:"name"`
  Path string `yaml:"path"`
}

type RunnerConfig struct {
  Log          *log.Logger
  WorkDir       string
  CacheDir      string
  Name          string
  Image         string
  CoreIds     []int
  Devices     []string
  Path          string
  Cmd           string
  AllowOverride bool
}

type Runner interface {
  Init()     error
  Run()    (int, error)
  Destroy()  error
}

// NewRunner returns the name of the 
func NewRunner(cfg *RunnerConfig) (Runner, error) {
  ref, err := ParseImageName(cfg.Image)
  if err != nil {
    return nil, err
  }

  if len(ref.Runtime) == 0 {
    ref.Runtime = DefaultRuntime
  }

  var runner Runner
  switch runtime := ref.Runtime; runtime {
	case "runc":
    runner = &RuncRunner{
      Config: cfg,
    }
	default:
    return nil, fmt.Errorf("Unsupported container runtime: %s", runtime)
	}

  err = runner.Init(cfg.Inputs, cfg.Outputs, dryRun)
  if err != nil {
    return nil, fmt.Errorf("Could not initialize runner: %s", err)
  }

  return runner, nil
}
