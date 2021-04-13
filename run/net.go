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
	"net"

	"github.com/lancs-net/netns/bridge"
	"github.com/lancs-net/netns/network"
	"github.com/opencontainers/runtime-spec/specs-go"

  "github.com/lancs-net/ukbench/log"
)

type Bridge struct {
  Name       string
  Interface  string
  Subnet     string
  CacheDir   string
	netOpt     network.Opt
	brOpt      bridge.Opt
	client    *network.Client
}

// Init prepares netns
func (b *Bridge) Init(dryRun bool) error {
  return nil
}

// Create a veth pair with the bridge for the container
func (b *Bridge) Create(s *specs.State) (net.IP, error) {
  // Create the bridge using netns
  b.netOpt.ContainerInterface = b.Interface
  b.netOpt.BridgeName = b.Name
  b.brOpt.Name = b.Name
  b.brOpt.IPAddr = b.Subnet
  b.netOpt.StateDir = b.CacheDir

  log.Debugf("Creating bridge %s...", b.Name)
  client, err := network.New(b.netOpt)
  if err != nil {
    return nil, err
  }
  
  return client.Create(s, b.brOpt, "")
}
