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
  "fmt"
  "errors"
)

const (
	// NameTotalLengthMax is the maximum total number of characters in a repository name.
	NameTotalLengthMax = 255
  // DefaultTag defines the default tag used when performing images related actions and no tag or digest is specified
  DefaultTag = "latest"
  // DefaultHostname is the default built-in hostname
  DefaultHostname = "docker.io"
  // LegacyDefaultHostname is automatically converted to DefaultHostname
  LegacyDefaultHostname = "index.docker.io"
  // DefaultRepoPrefix is the prefix used for default repositories in default host
  DefaultRepoPrefix = "library/"
)

var (
	// ErrReferenceInvalidFormat represents an error while trying to parse a string as a reference.
	ErrReferenceInvalidFormat = errors.New("invalid reference format")

	// ErrTagInvalidFormat represents an error while trying to parse a string as a tag.
	ErrTagInvalidFormat = errors.New("invalid tag format")

	// ErrDigestInvalidFormat represents an error while trying to parse a string as a tag.
	ErrDigestInvalidFormat = errors.New("invalid digest format")

	// ErrNameEmpty is returned for empty, invalid repository names.
	ErrNameEmpty = errors.New("repository name must have at least one component")

	// ErrNameTooLong is returned when a repository name is longer than NameTotalLengthMax.
	ErrNameTooLong = fmt.Errorf("repository name must not be more than %v characters", NameTotalLengthMax)
)

func Parse(s string) (Reference, error) {
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

	ref := reference{
		runtime: matches[2],
		name:    matches[3],
		tag:     matches[4],
	}

	if matches[5] != "" {
		var err error
		ref.digest, err = digest.ParseDigest(matches[5])
		if err != nil {
			return nil, err
		}
	}

	r := getBestReferenceType(ref)
	if r == nil {
		return nil, ErrNameEmpty
	}

	return r, nil
}
