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
