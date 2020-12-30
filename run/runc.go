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
  "os"
  "fmt"

  "golang.org/x/sys/unix"
  "github.com/opencontainers/runc/libcontainer"
  "github.com/opencontainers/runtime-spec/specs-go"
  "github.com/opencontainers/runc/libcontainer/specconv"
  "github.com/opencontainers/runc/libcontainer/configs"

  "github.com/lancs-net/ukbench/log"
)

var (
  defaultEnvironment = []string{
    "TERM=xterm",
    "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
  }
  defaultMountFlags = unix.MS_NOEXEC | unix.MS_NOSUID | unix.MS_NODEV
  defaultCapabilities = []string{
    "CAP_CHOWN",
    "CAP_DAC_OVERRIDE",
    "CAP_FSETID",
    "CAP_FOWNER",
    "CAP_MKNOD",
    "CAP_NET_RAW",
    "CAP_SETGID",
    "CAP_SETUID",
    "CAP_SETFCAP",
    "CAP_SETPCAP",
    "CAP_NET_BIND_SERVICE",
    "CAP_SYS_CHROOT",
    "CAP_KILL",
    "CAP_AUDIT_WRITE",
  }
)

type RuncRunner struct {
  log      *log.Logger
  Config   *RunnerConfig
  Bridge   *Bridge
  container libcontainer.Container 
}

func (r *RuncRunner) Init(in *[]Input, out *[]Output, dryRun bool) error {
  // Set the logger
  r.log = r.Config.Log
  
  // Download the image to the cache
  r.log.Infof("Pulling image: %s...", r.Config.Image)
  image, err := PullImage(r.Config.Image, r.Config.CacheDir)
  if err != nil {
    return fmt.Errorf("Could not download image: %s", err)
  }
  
  digest, err := image.Digest()
  if err != nil {
    return fmt.Errorf("Could not process digest: %s", err)
  }
  
  r.log.Debugf("Pulled: %s", digest)

  // Extract the image to the desired location
  r.log.Infof("Extracting image to: %s", r.Config.WorkDir)
  err = UnpackImage(image, r.Config.CacheDir, r.Config.WorkDir, r.Config.AllowOverride)
  if err != nil {
    return fmt.Errorf("Could not extract image: %s", err)
  }

  r.log.Debug("Initialising runc container...")

  factory, err := libcontainer.New(
    "/var/lib/container",
    libcontainer.Cgroupfs,
    libcontainer.InitArgs(os.Args[0], "runc-init"),
  )
  if err != nil {
    return err
  }

  var allowedDevices []*configs.DeviceRule
  for _, device := range specconv.AllowedDevices {
    allowedDevices = append(allowedDevices, &device.DeviceRule)
  }

  config := &configs.Config{
    Rootfs: r.Config.WorkDir,
    Capabilities: &configs.Capabilities{
      Bounding: defaultCapabilities,
      Effective: defaultCapabilities,
      Inheritable: defaultCapabilities,
      Permitted: defaultCapabilities,
      Ambient: defaultCapabilities,
    },
    Namespaces: configs.Namespaces([]configs.Namespace{
      {Type: configs.NEWNS},
      {Type: configs.NEWUTS},
      {Type: configs.NEWIPC},
      {Type: configs.NEWPID},
      {Type: configs.NEWUSER},
      {Type: configs.NEWNET},
      {Type: configs.NEWCGROUP},
    }),
    Cgroups: &configs.Cgroup{
      Name:      r.log.Prefix,
      Parent:    "",
      Resources: &configs.Resources{
        MemorySwappiness: nil,
        Devices:          allowedDevices,
      },
    },
    MaskPaths: []string{
      "/proc/kcore",
      "/sys/firmware",
    },
    ReadonlyPaths: []string{
      "/proc/sys", "/proc/sysrq-trigger", "/proc/irq", "/proc/bus",
    },
    Devices:  specconv.AllowedDevices,
    Hostname: r.log.Prefix,
    Mounts: []*configs.Mount{
      {
        Source:      "proc",
        Destination: "/proc",
        Device:      "proc",
        Flags:       defaultMountFlags,
      },
      {
        Source:      "tmpfs",
        Destination: "/dev",
        Device:      "tmpfs",
        Flags:       unix.MS_NOSUID | unix.MS_STRICTATIME,
        Data:        "mode=755",
      },
      {
        Source:      "devpts",
        Destination: "/dev/pts",
        Device:      "devpts",
        Flags:       unix.MS_NOSUID | unix.MS_NOEXEC,
        Data:        "newinstance,ptmxmode=0666,mode=0620,gid=5",
      },
      {
        Device:      "tmpfs",
        Source:      "shm",
        Destination: "/dev/shm",
        Data:        "mode=1777,size=65536k",
        Flags:       defaultMountFlags,
      },
      {
        Source:      "mqueue",
        Destination: "/dev/mqueue",
        Device:      "mqueue",
        Flags:       defaultMountFlags,
      },
      {
        Source:      "sysfs",
        Destination: "/sys",
        Device:      "sysfs",
        Flags:       defaultMountFlags | unix.MS_RDONLY,
      },
    },
    UidMappings: []configs.IDMap{
      {
        ContainerID: 0,
        HostID: 1000,
        Size: 65536,
      },
    },
    GidMappings: []configs.IDMap{
      {
        ContainerID: 0,
        HostID: 1000,
        Size: 65536,
      },
    },
    Networks: []*configs.Network{
      {
        Type:    "loopback",
        Address: "127.0.0.1/0",
        Gateway: "localhost",
      },
    },
    Rlimits: []configs.Rlimit{
      {
        Type: unix.RLIMIT_NOFILE,
        Hard: uint64(1025),
        Soft: uint64(1025),
      },
    },
    Hooks: configs.Hooks{
      configs.Prestart: configs.HookList{
        configs.NewFunctionHook(func(s *specs.State) error {
          ip, err := r.Bridge.Create(s)
          if err != nil {
            return err
          }

          r.log.Debugf("Container IP: %s\n", ip)

          return nil
        }),
      },
    },
  }

  // Attach each input as a mounted device to the container
  for _, input := range *in {
    // Parse the supplied flags
    var flags int

    // If no flags are specified, bind as RO
    if len(input.Options) == 0 {
      flags = unix.MS_BIND | unix.MS_RDONLY

    // Otherwise, parse the list of options
    } else {
      flags, _, _, _ = parseMountOptions(input.Options)
    }

    // Determine the input source's absolute path
    source, err := filepath.Abs(input.Source)
    if err != nil {
      r.log.Errorf("Cannot determine input source for %s: %s", input.Name, input.Source)
      continue
    }

    config.Mounts = append(config.Mounts, &configs.Mount{
      Device:      "bind",
      Source:      source,
      Destination: input.Destination,
      Flags:       flags,
    })
  }

  r.container, err = factory.Create(r.log.Prefix, config)
  if err != nil {
    return fmt.Errorf("Could not create container: %s", err)
  }

  return nil
}

// Run the runc container
func (r *RuncRunner) Run() (int, error) {
  if r.container == nil {
    return 1, fmt.Errorf("Cannot run container, missing initialization")
  }

  process := &libcontainer.Process{
    Args:   []string{"/bin/echo", "\"hello, world\""},
    Env:    []string{"PATH=/bin"},
    User:   "root",
    Stdout: r.log,
    Stderr: r.log,
  }

  err := r.container.Run(process)
  if err != nil {
    return 1, fmt.Errorf("Could not run container: %s", err)
  }

  // Wait for the process to finish
  state, err := process.Wait()
  if err != nil {
    return 1, fmt.Errorf("Could not wait for container to finish: %s", err)
  }

  return state.ExitCode(), nil
}

// Destroy the runc container
func (r *RuncRunner) Destroy() error {
  if r.container != nil {
    r.container.Destroy()
    r.container = nil
  }
  return nil
}

// parseMountOptions parses the string and returns the flags, propagation
// flags and any mount data that it contains.
func parseMountOptions(options []string) (int, []int, string, int) {
  var (
    flag     int
    pgflag   []int
    data     []string
    extFlags int
  )
  flags := map[string]struct {
    clear bool
    flag  int
  }{
    "acl":           {false, unix.MS_POSIXACL},
    "async":         {true, unix.MS_SYNCHRONOUS},
    "atime":         {true, unix.MS_NOATIME},
    "bind":          {false, unix.MS_BIND},
    "defaults":      {false, 0},
    "dev":           {true, unix.MS_NODEV},
    "diratime":      {true, unix.MS_NODIRATIME},
    "dirsync":       {false, unix.MS_DIRSYNC},
    "exec":          {true, unix.MS_NOEXEC},
    "iversion":      {false, unix.MS_I_VERSION},
    "lazytime":      {false, unix.MS_LAZYTIME},
    "loud":          {true, unix.MS_SILENT},
    "mand":          {false, unix.MS_MANDLOCK},
    "noacl":         {true, unix.MS_POSIXACL},
    "noatime":       {false, unix.MS_NOATIME},
    "nodev":         {false, unix.MS_NODEV},
    "nodiratime":    {false, unix.MS_NODIRATIME},
    "noexec":        {false, unix.MS_NOEXEC},
    "noiversion":    {true, unix.MS_I_VERSION},
    "nolazytime":    {true, unix.MS_LAZYTIME},
    "nomand":        {true, unix.MS_MANDLOCK},
    "norelatime":    {true, unix.MS_RELATIME},
    "nostrictatime": {true, unix.MS_STRICTATIME},
    "nosuid":        {false, unix.MS_NOSUID},
    "rbind":         {false, unix.MS_BIND | unix.MS_REC},
    "relatime":      {false, unix.MS_RELATIME},
    "remount":       {false, unix.MS_REMOUNT},
    "ro":            {false, unix.MS_RDONLY},
    "rw":            {true, unix.MS_RDONLY},
    "silent":        {false, unix.MS_SILENT},
    "strictatime":   {false, unix.MS_STRICTATIME},
    "suid":          {true, unix.MS_NOSUID},
    "sync":          {false, unix.MS_SYNCHRONOUS},
  }
  propagationFlags := map[string]int{
    "private":     unix.MS_PRIVATE,
    "shared":      unix.MS_SHARED,
    "slave":       unix.MS_SLAVE,
    "unbindable":  unix.MS_UNBINDABLE,
    "rprivate":    unix.MS_PRIVATE | unix.MS_REC,
    "rshared":     unix.MS_SHARED | unix.MS_REC,
    "rslave":      unix.MS_SLAVE | unix.MS_REC,
    "runbindable": unix.MS_UNBINDABLE | unix.MS_REC,
  }
  extensionFlags := map[string]struct {
    clear bool
    flag  int
  }{
    "tmpcopyup": {false, configs.EXT_COPYUP},
  }
  for _, o := range options {
    // If the option does not exist in the flags table or the flag
    // is not supported on the platform,
    // then it is a data value for a specific fs type
    if f, exists := flags[o]; exists && f.flag != 0 {
      if f.clear {
        flag &= ^f.flag
      } else {
        flag |= f.flag
      }
    } else if f, exists := propagationFlags[o]; exists && f != 0 {
      pgflag = append(pgflag, f)
    } else if f, exists := extensionFlags[o]; exists && f.flag != 0 {
      if f.clear {
        extFlags &= ^f.flag
      } else {
        extFlags |= f.flag
      }
    } else {
      data = append(data, o)
    }
  }
  return flag, pgflag, strings.Join(data, ","), extFlags
}
