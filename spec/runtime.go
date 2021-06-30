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

import (
  "time"
)

type RuntimeSpec struct {
  DryRun            bool          `short:"D" long:"dry-run"          yaml:"dry_run"          description:"Run without affecting the host or running the jobs."`
  AllowOverride     bool          `short:"O" long:"override"         yaml:"override"         description:"Override contents in directories (otherwise tasks allowed to fail)."`

  // Information output configuration
  Output            string        `short:"o" long:"output"           yaml:"output"           description:"The output channel to send printer output (valid output: stdout, file, tcp, udp)." default:"stdout"`
  Target            string        `          long:"target"           yaml:"target"           description:"For output 'file' the location, for 'tcp' or 'udp' the url (host:port) to the server."`

  // System host configuration
  CpuSets           string        `short:"s" long:"cpu-sets"         yaml:"cpu_sets"         description:"Specify which CPUs to run experiments on."`
  ProcFS            string        `          long:"procfs"           yaml:"procfs"           description:"Path to procfs." default:"/procfs"`
  WorkDir           string        `short:"w" long:"workdir"          yaml:"workdir"          description:"Specify working directory for outputting results, data, file systems, etc."`
  CacheDir          string        `          long:"cachedir"         yaml:"cachedir"         description:"Specify cache directory for temporary files, etc."`
  ResultsDir        string        `          long:"resultsdir"       yaml:"resultsdir"       description:"Specify directory for output files of results, etc."`
  LibvirtURI        string        `short:"l" long:"libvirt"          yaml:"libvirt"          description:"Connection URI to libvirt daemon." default:"qemu:///system"`
  ContainerdURI     string        `short:"c" long:"containerd"       yaml:"containerd"       description:"Connection URI to containerd's GRPC server." default:"/run/containerd/containerd.sock"`

  // Network configuration
  HostNetwork       string        `short:"h" long:"host-net"         yaml:"host_network"     description:"The host network to attach the bridge to." default:"eth0"`
  BridgeName        string        `short:"b" long:"bridge"           yaml:"bridge_name"      description:"The bridge network to attach services to." default:"ukbench0"`
  BridgeSubnet      string        `short:"n" long:"subnet"           yaml:"bridge_subnet"    description:"The subnet of addresses for container services." default:"172.88.0.1/16"`

  // Scheduler configuration      
  Scheduler         string        `short:"x" long:"scheduler"        yaml:"scheduler"        description:"Parameter permutation scheduler." default:"simple"`
  ScheduleGrace     int           `short:"g" long:"scheduler-grace"  yaml:"schedule_grace"   description:"Number of seconds to gracefully wait in the scheduler." default:"1"`
  MaxRetries        int           `short:"r" long:"max-retries"      yaml:"max_retries"      description:"Maximum number of retries for a failed permutation." default:"0"`

  // Metrics and measurement configuration
  MeasureCPU        bool          `          long:"measure-cpu"      yaml:"measure_cpu"      description:"Enable cpu metrics."`
  MeasureMEM        bool          `          long:"measure-mem"      yaml:"measure_mem"      description:"Enable memory metrics."`
  MeasureDISK       bool          `          long:"measure-disk"     yaml:"measure_disk"     description:"Enable disk metrics."`
  MeasureNET        bool          `          long:"measure-net"      yaml:"measure_net"      description:"Enable network metrics."`
  MeasureIO         bool          `          long:"measure-io"       yaml:"measure_io"       description:"Enable I/O metrics."`
  MeasurePressure   bool          `          long:"measure-pressure" yaml:"measure_pressure" description:"Enable pressure metrics (requires kernel 4.20+)."`
  MeasureHost       bool          `          long:"measure-host"     yaml:"measure_host"     description:"Enable host metrics."`
  MeasureAllMetrics bool          `short:"A" long:"measure-all"      yaml:"measure_all"      description:"Enable all metrics."`
  Frequency         time.Duration `short:"f" long:"frequency"        yaml:"frequency"        description:"Frequency (in seconds) for collecting metrics." default:"1"`

  NetworkDevice     string        `          long:"netdev"           yaml:"netdev"           description:"The network device used for the virtual traffic"`
	StorageDevice     string        `          long:"storedev"         yaml:"storedev"         description:"The storage device used for the virtual block devices"`
}
