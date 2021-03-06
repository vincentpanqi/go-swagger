// Copyright 2015 go-swagger maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package validate

import (
	"fmt"
	"strings"

	"github.com/go-openapi/errors"
)

// Defaulter ...
type Defaulter interface {
	Apply()
}

// DefaulterFunc ...
type DefaulterFunc func()

// Apply ...
func (f DefaulterFunc) Apply() {
	f()
}

// Result represents a validation result set
// Matchcount is used to determine
// which errors are relevant in the case of AnyOf, OneOf
// schema validation. Results from the validation branch
// with most matches get eventually selected.
// TODO keep path of key originating the error
type Result struct {
	Errors     []error
	Warnings   []error
	MatchCount int
	Defaulters []Defaulter
}

// Merge merges this result with the other one, preserving match counts etc
func (r *Result) Merge(others ...*Result) *Result {
	for _, other := range others {
		if other != nil {
			r.AddErrors(other.Errors...)
			r.AddWarnings(other.Warnings...)
			r.MatchCount += other.MatchCount
			r.Defaulters = append(r.Defaulters, other.Defaulters...)
		}
	}
	return r
}

// MergeAsErrors merges this result with the other one, preserving match counts etc.
// Warnings from input are merged as Errors in the returned merged Result.
func (r *Result) MergeAsErrors(others ...*Result) *Result {
	for _, other := range others {
		if other != nil {
			r.AddErrors(other.Errors...)
			r.AddErrors(other.Warnings...)
			r.MatchCount += other.MatchCount
			r.Defaulters = append(r.Defaulters, other.Defaulters...)
		}
	}
	return r
}

// MergeAsWarnings merges this result with the other one, preserving match counts etc.
// Errors from input are merged as Warnings in the returned merged Result.
func (r *Result) MergeAsWarnings(others ...*Result) *Result {
	for _, other := range others {
		if other != nil {
			r.AddWarnings(other.Errors...)
			r.AddWarnings(other.Warnings...)
			r.MatchCount += other.MatchCount
			r.Defaulters = append(r.Defaulters, other.Defaulters...)
		}
	}
	return r
}

// AddErrors adds errors to this validation result (if not already reported).
// Since the same check may be passed several times while exploring the
// spec structure (via $ref, ...) it is important to keep reported messages
// unique.
func (r *Result) AddErrors(errors ...error) {
	found := false
	for _, e := range errors {
		if e != nil {
			for _, isReported := range r.Errors {
				if e.Error() == isReported.Error() {
					found = true
					break
				}
			}
			if !found {
				r.Errors = append(r.Errors, e)
			}
		}
	}
}

// AddWarnings adds warnings to this validation result (if not already reported)
func (r *Result) AddWarnings(warnings ...error) {
	found := false
	for _, e := range warnings {
		if e != nil {
			for _, isReported := range r.Warnings {
				if e.Error() == isReported.Error() {
					found = true
					break
				}
			}
			if !found {
				r.Warnings = append(r.Warnings, e)
			}
		}
	}
}

// KeepRelevantErrors strips a result from standard errors and keeps
// the ones which are supposedly more accurate.
// The original result remains unaffected (creates a new instance of Result).
// We do that to work around the "matchCount" filter which would otherwise
// strip our result from some accurate error reporting from lower level validators.
//
// NOTE: this implementation with a placeholder (IMPORTANT!) is neither clean nor
// very efficient. On the other hand, relying on go-openapi/errors to manipulate
// codes would require to change a lot here. So, for the moment, let's go with
// placeholders.
func (r *Result) KeepRelevantErrors() *Result {
	strippedErrors := []error{}
	for _, e := range r.Errors {
		if strings.HasPrefix(e.Error(), "IMPORTANT!") {
			strippedErrors = append(strippedErrors, fmt.Errorf(strings.TrimPrefix(e.Error(), "IMPORTANT!")))
		}
	}
	strippedWarnings := []error{}
	for _, e := range r.Warnings {
		if strings.HasPrefix(e.Error(), "IMPORTANT!") {
			strippedWarnings = append(strippedWarnings, fmt.Errorf(strings.TrimPrefix(e.Error(), "IMPORTANT!")))
		}
	}
	strippedResult := new(Result)
	strippedResult.Errors = strippedErrors
	strippedResult.Warnings = strippedWarnings
	return strippedResult
}

// IsValid returns true when this result is valid
// TODO: allowing to work on nil object (same with HasErrors(), ...)
// same for Merge, etc... => would result in safe, more concise calls.
func (r *Result) IsValid() bool {
	return len(r.Errors) == 0
}

// HasErrors returns true when this result is invalid
func (r *Result) HasErrors() bool {
	return !r.IsValid()
}

// HasWarnings returns true when this result contains warnings
func (r *Result) HasWarnings() bool {
	return len(r.Warnings) > 0
}

// Inc increments the match count
func (r *Result) Inc() {
	r.MatchCount++
}

// AsError renders this result as an error interface
// TODO: reporting / pretty print with path ordered and indented
func (r *Result) AsError() error {
	if r.IsValid() {
		return nil
	}
	return errors.CompositeValidationError(r.Errors...)
}

// ApplyDefaults ...
func (r *Result) ApplyDefaults() {
	for _, d := range r.Defaulters {
		d.Apply()
	}
}
