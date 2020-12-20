package log
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

  "github.com/muesli/termenv"
)

// LogLevel is an enum-like type that we can use to designate the log level
type LogLevel int

const (
  DEBUG LogLevel = iota
  INFO
  WARNING
  ERROR
  FATAL
)

// Logger is a base struct that could eventually maintain connections to
// something like bugsnag or logging tools.
type Logger struct{
  LogLevel LogLevel
  Prefix   string
}

var (
  logger *Logger
)

func init() {
  logger = &Logger{
    LogLevel: INFO,
  }
}

// log is a private function that manages the internal logic about what and how
// to log data depending on the log level.
func (l *Logger) log(level LogLevel, format string, messages ...interface{}) {
  var logType string
  var logColor termenv.ANSIColor
  switch level {
  case DEBUG:
    logType = "DEBU"
    logColor = termenv.ANSICyan
    break
  case WARNING:
    logType = "WARN"
    logColor = termenv.ANSIYellow
    break
  case ERROR:
    logType = "ERRO"
    logColor = termenv.ANSIRed
    break
  case FATAL:
    logType = "FATA"
    logColor = termenv.ANSIRed
    break
  default:
    logType = "INFO"
    logColor = termenv.ANSIBlue
    break
  }

  if level < l.LogLevel {
    return
  }

  // Add some colours!
  out := termenv.String(logType)
  out = out.Foreground(logColor)

  if len(l.Prefix) > 0 {
    fmt.Printf("[%s][%s] %s\n", out, l.Prefix, fmt.Sprintf(format, messages...))
  } else {
    fmt.Printf("[%s] %s\n", out, fmt.Sprintf(format, messages...))
  }
}

func SetLevel(level LogLevel) {
  logger.LogLevel = level
}

func GetLevel() LogLevel {
  return logger.LogLevel
}

func Debug(messages ...interface{}) {
  logger.log(DEBUG, "%s", messages...)
}

func Debugf(format string, messages ...interface{}) {
  logger.log(DEBUG, format, messages...)
}

func (l *Logger) Debug(messages ...interface{}) {
  l.log(DEBUG, "%s", messages...)
}

func (l *Logger) Debugf(format string, messages ...interface{}) {
  l.log(DEBUG, format, messages...)
}

func Info(messages ...interface{}) {
  logger.log(INFO, "%s", messages...)
}

func Infof(format string, messages ...interface{}) {
  logger.log(INFO, format, messages...)
}

func (l *Logger) Info(messages ...interface{}) {
  l.log(INFO, "%s", messages...)
}

func (l *Logger) Infof(format string, messages ...interface{}) {
  l.log(INFO, format, messages...)
}
