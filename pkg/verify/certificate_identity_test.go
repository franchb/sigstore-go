// Copyright 2023 The Sigstore Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package verify

import (
	"testing"

	"github.com/franchb/sigstore-go/pkg/fulcio/certificate"
	"github.com/stretchr/testify/assert"
)

const (
	ActionsIssuerValue = "https://token.actions.githubusercontent.com"
	ActionsIssuerRegex = "githubusercontent.com$"
	SigstoreSanValue   = "https://github.com/sigstore/sigstore-js/.github/workflows/release.yml@refs/heads/main"
	SigstoreSanRegex   = "^https://github.com/sigstore/sigstore-js/"
)

func TestCertificateIdentityVerify(t *testing.T) {
	// given a certificate summary, it does what we expect

	actualCert := certificate.Summary{
		SubjectAlternativeName: SigstoreSanValue,
		Extensions: certificate.Extensions{
			Issuer:                              ActionsIssuerValue,
			GithubWorkflowTrigger:               "push",
			GithubWorkflowSHA:                   "f0b49a04e5a62250e0f60fb128004a73110fe311",
			GithubWorkflowName:                  "Release",
			GithubWorkflowRepository:            "sigstore/sigstore-js",
			GithubWorkflowRef:                   "refs/heads/main",
			BuildSignerURI:                      "https://github.com/sigstore/sigstore-js/.github/workflows/release.yml@refs/heads/main",
			BuildSignerDigest:                   "f0b49a04e5a62250e0f60fb128004a73110fe311",
			RunnerEnvironment:                   "github-hosted",
			SourceRepositoryURI:                 "https://github.com/sigstore/sigstore-js",
			SourceRepositoryDigest:              "f0b49a04e5a62250e0f60fb128004a73110fe311",
			SourceRepositoryRef:                 "refs/heads/main",
			SourceRepositoryIdentifier:          "495574555",
			SourceRepositoryOwnerURI:            "https://github.com/sigstore",
			SourceRepositoryOwnerIdentifier:     "71096353",
			BuildConfigURI:                      "https://github.com/sigstore/sigstore-js/.github/workflows/release.yml@refs/heads/main",
			BuildConfigDigest:                   "f0b49a04e5a62250e0f60fb128004a73110fe311",
			BuildTrigger:                        "push",
			RunInvocationURI:                    "https://github.com/sigstore/sigstore-js/actions/runs/5904696764/attempts/1",
			SourceRepositoryVisibilityAtSigning: "public",
		},
	}

	// First, let's test happy paths:
	issuerOnlyID, _ := certIDForTesting("", "", ActionsIssuerValue, "", "")
	assert.NoError(t, issuerOnlyID.Verify(actualCert))

	issuerOnlyRegex, _ := certIDForTesting("", "", "", ActionsIssuerRegex, "")
	assert.NoError(t, issuerOnlyRegex.Verify(actualCert))

	sanValueOnly, _ := certIDForTesting(SigstoreSanValue, "", "", "", "")
	assert.NoError(t, sanValueOnly.Verify(actualCert))

	sanRegexOnly, _ := certIDForTesting("", SigstoreSanRegex, "", "", "")
	assert.NoError(t, sanRegexOnly.Verify(actualCert))

	// multiple values can be specified
	sanRegexAndIssuer, _ := certIDForTesting("", SigstoreSanRegex, ActionsIssuerValue, "", "github-hosted")
	assert.NoError(t, sanRegexAndIssuer.Verify(actualCert))

	// unhappy paths:
	// wrong issuer
	sanRegexAndWrongIssuer, _ := certIDForTesting("", SigstoreSanRegex, "https://token.actions.example.com", "", "")
	errValueMismatch := &ErrValueMismatch{}
	assert.ErrorAs(t, sanRegexAndWrongIssuer.Verify(actualCert), &errValueMismatch)
	assert.Equal(t, "expected issuer value \"https://token.actions.example.com\", got \"https://token.actions.githubusercontent.com\"", errValueMismatch.Error())

	// bad san regex
	badRegex, _ := certIDForTesting("", "^badregex.*", "", "", "")
	errValueRegexMismatch := &ErrValueRegexMismatch{}
	assert.ErrorAs(t, badRegex.Verify(actualCert), &errValueRegexMismatch)
	assert.Equal(t, "expected SAN value to match regex \"^badregex.*\", got \"https://github.com/sigstore/sigstore-js/.github/workflows/release.yml@refs/heads/main\"", errValueRegexMismatch.Error())

	// bad issuer regex
	badIssuerRegex, _ := certIDForTesting("", "", "", "^badregex$", "")
	assert.Error(t, badIssuerRegex.Verify(actualCert))

	// if we have an array of certIDs, only one needs to match
	ci, err := CertificateIdentities{sanRegexAndWrongIssuer, sanRegexAndIssuer}.Verify(actualCert)
	assert.NoError(t, err)
	assert.Equal(t, *ci, sanRegexAndIssuer)

	// if none match, we fail
	ci, err = CertificateIdentities{badRegex, sanRegexAndWrongIssuer}.Verify(actualCert)
	assert.Error(t, err)
	assert.Equal(t, "no matching CertificateIdentity found, last error: expected issuer value \"https://token.actions.example.com\", got \"https://token.actions.githubusercontent.com\"", err.Error())
	assert.Nil(t, ci)
	// test err unwrap for previous error
	errValueMismatch = &ErrValueMismatch{}
	assert.ErrorAs(t, err, &errValueMismatch)
	assert.Equal(t, "expected issuer value \"https://token.actions.example.com\", got \"https://token.actions.githubusercontent.com\"", errValueMismatch.Error())

	// if no certIDs are specified, we fail
	_, err = CertificateIdentities{}.Verify(actualCert)
	assert.Error(t, err)
	assert.Equal(t, "no matching CertificateIdentity found", err.Error())
}

func TestThatCertIDsAreFullySpecified(t *testing.T) {
	_, err := NewShortCertificateIdentity("", "", "", "")
	assert.Error(t, err)

	_, err = NewShortCertificateIdentity("foobar", "", "", "")
	assert.Error(t, err)

	_, err = NewShortCertificateIdentity("", ActionsIssuerRegex, "", "")
	assert.Error(t, err)

	_, err = NewShortCertificateIdentity("", "", "", SigstoreSanRegex)
	assert.Error(t, err)

	_, err = NewShortCertificateIdentity("foobar", "", "", SigstoreSanRegex)
	assert.Nil(t, err)

	_, err = NewShortCertificateIdentity("", ActionsIssuerRegex, "", SigstoreSanRegex)
	assert.Nil(t, err)
}

func certIDForTesting(sanValue, sanRegex, issuer, issuerRegex, runnerEnv string) (CertificateIdentity, error) {
	san, err := NewSANMatcher(sanValue, sanRegex)
	if err != nil {
		return CertificateIdentity{}, err
	}

	issuerMatcher, err := NewIssuerMatcher(issuer, issuerRegex)
	if err != nil {
		return CertificateIdentity{}, err
	}

	return CertificateIdentity{SubjectAlternativeName: san, Issuer: issuerMatcher, Extensions: certificate.Extensions{RunnerEnvironment: runnerEnv}}, nil
}
