package runtime
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
  "io/ioutil"

  "github.com/lancs-net/ukbench/internal/log"
)

type ProcValue struct {
  Path     string
  Original string
  Current  string
}

type Proc struct {
  Items []ProcValue
}

var procfs Proc

// remember adds new item to our stateful procfs so we can revert later
func (p *Proc) remember(path string, original string, new string) {
  for i, item := range p.Items {
    // Check if we're not remembering the reverse
    if item.Path == path && item.Original == new && item.Current == original {
      return
    }

    // Check if we're not overriding the original original
    if item.Path == path {
      original = item.Original

      // Delete existing entry
      p.Items = append(p.Items[:i], p.Items[i+1:]...)
    }
  }

  p.Items = append(p.Items, ProcValue{
    Path:     path,
    Original: original,
    Current:  new,
  })
}

// setProcfsValue sets a string value at a procfs path
func setProcfsValue(path string, value string, dryRun bool) error {
  // Check if the path is set
  if len(path) == 0 {
    return fmt.Errorf("File path cannot be empty")
  }

  // Check if the file exists
  stat, err := os.Stat(path); 
  if os.IsNotExist(err) {
    return fmt.Errorf("File does not exist: %s", path)
  }

  // Check if this file receives input via stdin
  if stat.Size() == 0 {
    log.Infof("Setting %s to %s", path, value)

  // This is a regular proc file with a set value
  } else {
    dat, err := ioutil.ReadFile(path)
    if err != nil {
      if dryRun {
        log.Warnf("Could not read file: %s", err)
      } else {
        return fmt.Errorf("Could not read file: %s", err)
      }
    }

    // Remove trailing \n if it exists
    current := strings.TrimSuffix(string(dat), "\n")

    // No need to set identical value
    if current == value {
      return nil
    }

    log.Infof("Setting %s from %s to %s", path, current, value)

    // Save the current value for later reset
    procfs.remember(path, current, value)
  }
  
  // Open file
  f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
  if err != nil {
    if dryRun {
      log.Warn(err)
    } else {
      return err
    }
  }
  
  if !dryRun {
    // Write new value to path
    _, err = fmt.Fprintf(f, "%s", value)
    if err != nil {
      return err
    }

    // Close file
    if err := f.Close(); err != nil {
      return err
    }
  }

  return nil
}

func PrepareEnvironment(cpus []int, dryRun bool) error {
  if _, err := os.Stat("/proc/self/ns/user"); os.IsNotExist(err) {
		return fmt.Errorf("userns is unsupported")
	}

  /*
   * Filesystem preparation
   */

  err := setProcfsValue("/proc/sys/fs/file-max", "20000", dryRun)
  if err != nil {
    return err
  }

  // Drop fs cache
  // err = setProcfsValue("/proc/sys/vm/drop_caches", "3", dryRun)
  // if err != nil {
  //   return err
  // }

  /*
   * Networking preparation
   */

  err = setProcfsValue("/proc/sys/net/core/somaxconn", "1024", dryRun)
  if err != nil {
    return err
  }

  err = setProcfsValue("/proc/sys/net/ipv4/ip_forward", "1", dryRun)
  if err != nil {
    return err
  }

  err = setProcfsValue("/proc/sys/net/ipv4/ip_local_port_range", "1024   60999", dryRun)
  if err != nil {
    return err
  }

  // err = setProcfsValue("/proc/sys/net/ipv4/tcp_tw_reusee", "1", dryRun)
  // if err != nil {
  //   return err
  // }

  err = setProcfsValue("/proc/sys/net/ipv4/tcp_keepalive_time", "60", dryRun)
  if err != nil {
    return err
  }
  
  err = setProcfsValue("/proc/sys/net/ipv4/tcp_keepalive_intvl", "60", dryRun)
  if err != nil {
    return err
  }

  /*
   * Processor preparation
   */
  // Set scaling governor performance
  for _, c := range cpus {
    err = setProcfsValue(fmt.Sprintf("/sys/devices/system/cpu/cpu%d/cpufreq/scaling_governor", c), "performance", dryRun)
    if err != nil {
      log.Warnf("Cannot set scaling governor: %s", err)
    }

    // err = setProcfsValue(fmt.Sprintf("/sys/devices/system/cpu/cpu%d/online", c), "0", dryRun)
    // if err != nil {
    //   log.Warnf("Cannot set scaling governor: %s", err)
    // }
  }
  
  // Disable Intel Turbo mode
  // TODO: Determine if this setting is even possible on this machine
  err = setProcfsValue("/sys/devices/system/cpu/intel_pstate/no_turbo", "1", dryRun)
  if err != nil {
    log.Warnf("Cannot set Intel Turbo mode: %s", err)
  }

  /*
   * Memory prepraration
   */

  // Disable ASLR
  err = setProcfsValue("/proc/sys/kernel/randomize_va_space", "0", dryRun)
  if err != nil {
    return err
  }

  /*
   * Namespacing
   */

  // Allow rootless containers
  err = setProcfsValue("/proc/sys/kernel/unprivileged_userns_clone", "1", dryRun)
  if err != nil {
    return err
  }

  return nil
}

// RevertEnvironment sets original Procfs entries 
func RevertEnvironment(dryRun bool) error {
  // Reset updated procfs itemss
  for _, item := range procfs.Items {
    err := setProcfsValue(item.Path, item.Original, dryRun)
    if err != nil {
      log.Warn(err)
    }
  }

  return nil
}
