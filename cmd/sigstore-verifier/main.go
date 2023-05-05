package main

import (
	"flag"
	"fmt"
	"os"

	prototrustroot "github.com/sigstore/protobuf-specs/gen/pb-go/trustroot/v1"
	protoverification "github.com/sigstore/protobuf-specs/gen/pb-go/verification/v1"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/github/sigstore-verifier/pkg/bundle"
	"github.com/github/sigstore-verifier/pkg/policy"
	"github.com/github/sigstore-verifier/pkg/root"
)

var requireTSA *bool
var trustedrootJSONpath *string

func init() {
	requireTSA = flag.Bool("requireTSA", false, "Require RFC 3161 signed timestamp")
	trustedrootJSONpath = flag.String("trustedrootJSONpath", "", "Path to trustedroot JSON file")
	flag.Parse()
	if flag.NArg() == 0 {
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Printf("Usage: %s [OPTIONS] BUNDLE_FILE ...\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	b, err := bundle.LoadJSONFromPath(flag.Arg(0))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if !*requireTSA && *trustedrootJSONpath != "" {
		err = policy.VerifyKeyless(b)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	} else {
		var tr *root.TrustedRoot

		if *trustedrootJSONpath != "" {
			trustedrootJSON, err := os.ReadFile(*trustedrootJSONpath)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			pbTrustedRoot := &prototrustroot.TrustedRoot{}
			err = protojson.Unmarshal(trustedrootJSON, pbTrustedRoot)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			tr, err = root.NewTrustedRootFromProtobuf(pbTrustedRoot)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		} else {
			tr, err = root.GetSigstoreTrustedRoot()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		}

		opts := &protoverification.ArtifactVerificationOptions{
			Signers: nil,
			TlogOptions: &protoverification.ArtifactVerificationOptions_TlogOptions{
				Threshold:                 0,
				PerformOnlineVerification: false,
				Disable:                   true,
			},
			CtlogOptions: &protoverification.ArtifactVerificationOptions_CtlogOptions{
				Threshold:   0,
				DetachedSct: false,
				Disable:     true,
			},
			TsaOptions: &protoverification.ArtifactVerificationOptions_TimestampAuthorityOptions{
				Threshold: 1,
				Disable:   false,
			},
		}

		p := policy.NewPolicy(tr, opts)
		err = p.VerifyPolicy(b)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	fmt.Println("Verification successful!")
}