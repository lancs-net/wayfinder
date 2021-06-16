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
  "fmt"
  "math"
  "strconv"

  "github.com/lancs-net/ukbench/internal/log"
)

type Param struct {
  Name           string `yaml:"name"`
  Type           string `yaml:"type"`
  Default        string `yaml:"default"`
  Only         []string `yaml:"only"`
  Min            string `yaml:"min"`
  Max            string `yaml:"max"`
  Step           string `yaml:"step"`
  StepMode       string `yaml:"step_mode"`
}

type ParamPermutation struct {
  Name           string
  Type           string
  Value          string
}

// parseParamInt attends to string parameters and its possible permutations
func parseParamStr(param *Param) ([]ParamPermutation, error) {
  var params []ParamPermutation

  if len(param.Only) > 0 {
    for _, val := range param.Only {
      params = append(params, ParamPermutation{
        Name:  param.Name,
        Type:  param.Type,
        Value: val,
      })
    }
  } else if len(param.Default) > 0 {
    params = append(params, ParamPermutation{
      Name:  param.Name,
      Type:  param.Type,
      Value: param.Default,
    })
  }

  return params, nil
}

// parseParamInt attends to integer parameters and its possible permutations
func parseParamInt(param *Param) ([]ParamPermutation, error) {
  var params []ParamPermutation

  // Parse values in only
  if len(param.Only) > 0 {
    for _, val := range param.Only {
      params = append(params, ParamPermutation{
        Name:  param.Name,
        Type:  param.Type,
        Value: val,
      })
    }

  // Parse range between min and max
  } else if len(param.Min) > 0 {
    min, err := strconv.Atoi(param.Min)
    if err != nil {
      return nil, err
    }
    
    max, err := strconv.Atoi(param.Max)
    if err != nil {
      return nil, err
    }

    if max < min {
      return nil, fmt.Errorf(
        "Min can't be greater than max for %s: %d < %d", param.Name, min, max,
      )
    }

    // Figure out the step
    step := 1
    if len(param.Step) > 0 {
      step, err = strconv.Atoi(param.Step)
      if err != nil || step == 0 {
        return nil, fmt.Errorf(
          "Invalid step for %s: %s", param.Name, param.Step,
        )
      }
    }

    // Use iterative step
    if len(param.StepMode) == 0 || param.StepMode == "increment" {
      for i := min; i <= max; i += step {
        params = append(params, ParamPermutation{
          Name:  param.Name,
          Type:  param.Type,
          Value: strconv.Itoa(i),
        })
      }

    // Use exponential step
    } else if param.StepMode == "power" {
      for i, j := min, min; i <= max; j++ {
        params = append(params, ParamPermutation{
          Name:  param.Name,
          Type:  param.Type,
          Value: strconv.Itoa(i),
        })
        i = int(math.Pow(float64(step), float64(j)))
      }

    // Unknown step mode
    } else {
      return nil, fmt.Errorf(
        "Unknown step mode for param %s: %s", param.Name, param.StepMode,
      )
    }

  } else if len(param.Default) > 0 {
    params = append(params, ParamPermutation{
      Name:  param.Name,
      Type:  param.Type,
      Value: param.Default,
    })

  } else {
    log.Warnf("Parameter not parsed: %s", param.Name)
  }

  return params, nil
}

// paramPermutations discovers all the possible variants of a particular
// parameter based on its type and options.
func paramPermutations(param *Param) ([]ParamPermutation, error) {
  switch t := param.Type; t {
  case "string":
    return parseParamStr(param)
  case "int":
    return parseParamInt(param)
  case "integer":
    return parseParamInt(param)
  }
  return nil, fmt.Errorf(
    "Unknown parameter type: \"%s\" for %s", param.Type, param.Name,
  )
}
