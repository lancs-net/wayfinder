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
  "io"
  "fmt"
  "crypto/md5"
)

type JobSpec struct {
  Params       []ParamSpec   `yaml:"params"`
  Inputs       []InputSpec   `yaml:"inputs"`
  Outputs      []OutputSpec  `yaml:"outputs"`
  Runs         []RunSpec     `yaml:"runs"`
  permutations []*JobPermutation
}

type JobPermutation struct {
  Params       []ParamPermutation
  Inputs      *[]InputSpec
  Outputs     *[]OutputSpec
  Runs        *[]RunSpec
  uuid          string
}

// next recursively iterates across paramters to generate a set of tasks
func (j *JobSpec) next(i int, perms []*JobPermutation, curr []ParamPermutation) ([]*JobPermutation, error) {
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
      var p = make([]ParamPermutation, len(j.Params))
      copy(p, curr)
      perm := &JobPermutation{
        Params:   p,
        Inputs:  &j.Inputs,
        Outputs: &j.Outputs,
        Runs:    &j.Runs,
      }
      perms = append(perms, perm)

    // Otherwise, recursively parse parameters in-order    
    } else {
      nextPerms, err := j.next(i + 1, nil, curr)
      if err != nil {
        return nil, err
      }

      perms = append(perms, nextPerms...)
    }
  }

  return perms, nil
}

// Permutations returns a list of all possible tasks based on parameterisation
func (j *JobSpec) Permutations() ([]*JobPermutation, error) {
  if j.permutations != nil {
    return j.permutations, nil
  }

  var perm []*JobPermutation

  perm, err := j.next(0, perm, nil)
  if err != nil {
    return nil, err
  }

  j.permutations = perm

  return perm, nil
}

func (jp *JobPermutation) UUID() string {
  if len(jp.uuid) == 0 {

    // Calculate the UUID based on a reproducible md5 seed
    md5val := md5.New()
    for _, param := range jp.Params {
      io.WriteString(md5val, fmt.Sprintf("%s=%s\n", param.Name, param.Value))
    }

    jp.uuid = fmt.Sprintf("%x", md5val.Sum(nil))
  }

  return jp.uuid
}
