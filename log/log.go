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
  "strings"

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
  SUCCESS
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
  case SUCCESS:
    logType = " :) "
    logColor = termenv.ANSIGreen
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

func Warn(messages ...interface{}) {
  logger.log(WARNING, "%s", messages...)
}

func Warnf(format string, messages ...interface{}) {
  logger.log(WARNING, format, messages...)
}

func (l *Logger) Warn(messages ...interface{}) {
  l.log(WARNING, "%s", messages...)
}

func (l *Logger) Warnf(format string, messages ...interface{}) {
  l.log(WARNING, format, messages...)
}

func Error(messages ...interface{}) {
  logger.log(ERROR, "%s", messages...)
}

func Errorf(format string, messages ...interface{}) {
  logger.log(ERROR, format, messages...)
}

func (l *Logger) Error(messages ...interface{}) {
  l.log(ERROR, "%s", messages...)
}

func (l *Logger) Errorf(format string, messages ...interface{}) {
  l.log(ERROR, format, messages...)
}

func Fatal(messages ...interface{}) {
  logger.log(FATAL, "%s", messages...)
}

func Fatalf(format string, messages ...interface{}) {
  logger.log(FATAL, format, messages...)
}

func (l *Logger) Fatal(messages ...interface{}) {
  l.log(FATAL, "%s", messages...)
}

func (l *Logger) Fatalf(format string, messages ...interface{}) {
  l.log(FATAL, format, messages...)
}

func Success(messages ...interface{}) {
  logger.log(SUCCESS, "%s", messages...)
}

func Successf(format string, messages ...interface{}) {
  logger.log(SUCCESS, format, messages...)
}

func (l *Logger) Success(messages ...interface{}) {
  l.log(SUCCESS, "%s", messages...)
}

func (l *Logger) Successf(format string, messages ...interface{}) {
  l.log(SUCCESS, format, messages...)
}

// Write implements io.Writer
func (l *Logger) Write(b []byte) (n int, err error) {
  if len(string(b)) > 0 {
    messages := strings.Split(string(b), "\n")
    for _, message := range messages {
      if len(message) > 0 {
        l.log(INFO, "%s", message)
      }
    }
  }
  return len(b), nil
}
