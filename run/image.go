package run
// SPDX-License-Identifier: Apache-2.0
//
// Copyright (C) 2015-2017 Thomas LE ROUX <thomas@leroux.io>
//               2020      Alexander Jung <a.jung@lancs.ac.uk>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Parse parses s and returns a syntactically valid Reference.
// If an error was encountered it is returned, along with a nil Reference.
// NOTE: Parse will not handle short digests.

import (
  "os"
  "fmt"
  "errors"
  "strings"
  "runtime"

  "github.com/tidwall/gjson"
	"github.com/google/go-containerregistry/pkg/crane"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

const (
  // DefaultRuntime is the runtime to use when not specified.
  DefaultRuntime = "runc"
  // NameTotalLengthMax is the maximum total number of characters in a
  // repository name.
  NameTotalLengthMax = 255
  // DefaultTag defines the default tag used when performing images related
  // actions and no tag or digest is specified.
  DefaultTag = "latest"
  // DefaultHostname is the default built-in hostname
  DefaultHostname = "docker.io"
  // LegacyDefaultHostname is automatically converted to DefaultHostname
  LegacyDefaultHostname = "index.docker.io"
  // DefaultRepoPrefix is the prefix used for default repositories in default
  // host.
  DefaultRepoPrefix = "library/"
)

var (
  // ErrReferenceInvalidFormat represents an error while trying to parse a
  // string as a reference.
  ErrReferenceInvalidFormat = errors.New("invalid reference format")
  // ErrTagInvalidFormat represents an error while trying to parse a string as a
  // tag.
  ErrTagInvalidFormat = errors.New("invalid tag format")
  // ErrDigestInvalidFormat represents an error while trying to parse a string
  // as a tag.
  ErrDigestInvalidFormat = errors.New("invalid digest format")
  // ErrNameEmpty is returned for empty, invalid repository names.
  ErrNameEmpty = errors.New("repository name must have at least one component")
  // ErrNameTooLong is returned when a repository name is longer than
  // NameTotalLengthMax.
  ErrNameTooLong = fmt.Errorf("repository name must not be more than %v characters", NameTotalLengthMax)
)

// Image is an object with a full name
type Image struct {
  // Runtime is the normalized name of the runtime service, e.g. "docker"
  Runtime    string
  // Name is the normalized repository name, like "ubuntu".
  Name       string
  // String is the full reference, like "ubuntu@sha256:abcdef..."
  String     string
  // FullName is the full repository name with hostname, like "docker.io/library/ubuntu"
  FullName   string
  // Hostname is the hostname for the reference, like "docker.io"
  Hostname   string
  // RemoteName is the the repository component of the full name, like "library/ubuntu"
  RemoteName string
  // Tag is the tag of the image, e.g. "latest"
  Tag        string
}

func ParseImageName(s string) (*Image, error) {
  matches := ReferenceRegexp.FindStringSubmatch(s)
  if matches == nil {
    if s == "" {
      return nil, ErrNameEmpty
    }
    // TODO(dmcgowan): Provide more specific and helpful error
    return nil, ErrReferenceInvalidFormat
  }

  if len(matches[2]) > NameTotalLengthMax {
    return nil, ErrNameTooLong
  }

  image := &Image{
    Runtime: matches[2],
    Name:    matches[3],
    Tag:     matches[4],
  }

  return image, nil
}

// PullImage downloads an image
func PullImage(image, cacheDir string) (v1.Image, error) {
  var options []crane.Option

  // options = append(options, crane.Insecure)

  // Use current built OS and architecture
  options = append(options, crane.WithPlatform(&v1.Platform{
    OS: runtime.GOOS,
    Architecture: runtime.GOARCH,
  }))

  // Grab the remote manifest
  manifest, err := crane.Manifest(image, options...)
  if err != nil {
    return nil, fmt.Errorf("failed fetching manifest for %s: %v", image, err)
  }

  if !gjson.Valid(string(manifest)) {
    return nil, fmt.Errorf("Cannot parse manifest: %s", string(manifest))
  }

  value := gjson.Get(string(manifest), "config.digest").Value().(string)
  if value == "" {
    return nil, fmt.Errorf("Malformed manifest: %s", string(manifest))
  }
  
  digest := strings.Split(value, ":")[1]
  tarball := fmt.Sprintf("%s/%s.tar.gz", cacheDir, digest)

  // Download the tarball of the image if not available in the cache
  if _, err := os.Stat(tarball); os.IsNotExist(err) {
    // Create the cacheDir if it does not already exist
    if cacheDir != "" {
      if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
        os.MkdirAll(cacheDir, os.ModePerm)
      }
    }
    
    // Pull the image
    img, err := crane.Pull(image, options...)
    if err != nil {
      return nil, fmt.Errorf("Could not pull image: %s", err)
    }
    
    f, err := os.Create(tarball)
    if err != nil {
      return nil, fmt.Errorf("Failed to open %s: %v", tarball, err)
    }
  
    defer f.Close()
  
    err = crane.Save(img, image, tarball)
    if err != nil {
      return nil, fmt.Errorf("Could not save image: %s", err)
    }
  }

  img, err := crane.Load(tarball)
  if err != nil {
    return nil, fmt.Errorf("Could not load image: %s", err)
  }

  return img, nil
}
