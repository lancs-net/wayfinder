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

  "github.com/lancs-net/wayfinder/log"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use: "wayfinder",
	Short: `wayfinder: OS Configuration Micro-Benchmarking Framework`,
	PersistentPreRun: doRootCmd,
	DisableFlagsInUseLine: true,
}

// Execute adds all child commands to the root command and sets flags
// appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// doRootCmd 
func doRootCmd(cmd *cobra.Command, args []string) {
	verbose, err := cmd.Flags().GetBool("verbose")
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(0)
	}

	initLogging(verbose)
}

func init() {
	// Persistent global flags
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose logging")

	// Subcommands
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(versionCmd)
  rootCmd.AddCommand(runcInitCmd)
}

// initLogging prepares logrus with sensible defaults
func initLogging(verbose bool) {
  // Only log the warning severity or above.
  if verbose {
		log.SetLevel(log.DEBUG)
	}
}
