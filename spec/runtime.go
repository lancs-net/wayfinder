package spec
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

type RuntimeSpec struct {
  CpuSets       string  `short:"s" long:"cpu-sets" yaml:"cpu_sets" description:"Specify which CPUs to run experiments on."`
  DryRun        bool    `short:"D" long:"dry-run" yaml:"dry_run" description:"Run without affecting the host or running the jobs." default:"false"`
  ScheduleGrace int     `short:"g" long:"scheduler-grace" yaml:"schedule_grace" description:"Number of seconds to gracefully wait in the scheduler." default:"1"`
  WorkDir       string  `short:"w" long:"workdir" yaml:"workdir" description:"Specify working directory for outputting results, data, file systems, etc."`
  AllowOverride bool    `short:"O" long:"override" yaml:"allow_io_override" description:"Override contents in directories (otherwise tasks allowed to fail)." default:"false"`
  HostNetwork   string  `short:"h" long:"host-net" yaml:"host_network" description:"" default:""`
  BridgeName    string  `short:"b" long:"bridge" yaml:"bridge_name" description:"" default:"ukbench0"`
  BridgeSubnet  string  `short:"n" long:"subnet" yaml:"bridge_subnet" description:"" default:"172.88.0.1/16"`
  MaxRetries    int     `short:"r" long:"max-retries" yaml:"max_retries" description:"Maximum number of retries for a failed permutation" default:"0"`
  Scheduler     string  `short:"x" long:"scheduler" yaml:"scheduler" description:"Parameter permutation scheduler" default:"simple"`
}
