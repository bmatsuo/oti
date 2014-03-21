// Copyright 2014, Bryan Matsuo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// launch.go [created: Thu, 20 Mar 2014]

package main

import (
	"code.google.com/p/go-uuid/uuid"
	"github.com/bmatsuo/oti/otisub"
	_ "github.com/crowdmob/goamz/aws"
	awsec2 "github.com/crowdmob/goamz/ec2"

	"flag"
	"fmt"
	"strconv"
	"strings"
)

var launch = otisub.Register("launch", func(args []string) {
	fs := otisub.FlagSet(flag.ExitOnError, "inspect", "imagename [directive ...] ...")
	waitPending := fs.Bool("w", false, "wait while instances are 'pending'")
	fs.Parse(args)
	args = fs.Args()

	mfts, err := ParseUserLaunchManifest(args)
	if err != nil {
		Log.Fatal(err)
	}

	if len(mfts) == 0 {
		Log.Fatal("no manifests")
	}

	for _, m := range mfts {
		Log.Printf("%#v", m)
	}

	sessionId := uuid.New()
	Log.Println("oti session id: ", sessionId)

	// find images by tag co.bmats.oti.ID
	// launch instances

	// wait for instances to boot
	if *waitPending {
		Log.Fatal("waiting not implemented")
	}

	fmt.Println("LAUNCH!")
})

// User Launch Manifest -- information read from the command line
type ULM struct {
	Name         string // OTI name that can be used to filter instances
	IdentityFile string // SSH .pem file
	Min, Max     int    // may not be empty
}

type ArgumentError struct {
	i   int
	err error
}

func (err ArgumentError) Error() string {
	return fmt.Sprintf("argument %d: %v", err.i, err.err)
}

var ErrEndOfArgs = ArgumentError{-1, fmt.Errorf("no more arguments")}

// parses a set launch manifest. manifests have the form
//	name [ flag[=val] ... ] -- ...
// for reference use the following list of flags and the default values
//	flag      alias  default
//	min              1
//	max              1
//	identity  i      ""
func ParseUserLaunchManifest(args []string) ([]ULM, error) {
	ulms := make([]ULM, 0, len(args))
	sepseq := "--"

	parseUlm := func(args []string) (ulm ULM, rest []string, err error) {
		rest = args

		if len(rest) == 0 {
			return ULM{}, nil, ErrEndOfArgs
		}

		if rest[0] == sepseq {
			err := fmt.Errorf("unexpected separator sequence %v", sepseq)
			return ULM{}, nil, err
		}

		ulm.Name, rest = args[0], rest[1:]

		// set defaults
		ulm.Min, ulm.Max = 1, 1

		if len(rest) == 0 {
			return ulm, nil, nil
		}

		retErr := func(err error) (ULM, []string, error) {
			return ULM{}, nil, err
		}
		ulmErr := func(err error) error { return fmt.Errorf("%v %v", ulm.Name, err) }
		ulmFlagErr := func(key string, err error) error {
			return ulmErr(fmt.Errorf("invalid flag %q: %v", key, err))
		}

		flags := make(map[string][]string)
		for len(rest) > 0 && rest[0] != sepseq {
			var head string
			head, rest = rest[0], rest[1:]
			key, value := head, ""

			pair := strings.SplitN(key, "=", 2)
			if len(pair) == 2 {
				key, value = pair[0], pair[1]
			}

			switch key {
			case "min", "max":
			case "identity", "i":
				key = "identity" // alias
			default:
				err := fmt.Errorf("unexpected flag %v", key)
				return retErr(ulmErr(err))
			}

			flags[key] = append(flags[key], value)
		}

		for k, vs := range flags {
			var err error
			numvs := len(vs)
			switch k {
			case "min":
				if numvs > 1 {
					err = fmt.Errorf("specified multiple times")
				} else {
					ulm.Min, err = strconv.Atoi(vs[0])
				}
			case "max":
				if numvs > 1 {
					err = fmt.Errorf("specified multiple times")
				} else {
					ulm.Max, err = strconv.Atoi(vs[0])
				}
			case "identity":
				if numvs > 1 {
					err = fmt.Errorf("specified multiple times")
				} else {
					ulm.IdentityFile = vs[0]
				}
			}
			if err != nil {
				return retErr(ulmFlagErr(k, err))
			}
		}

		if ulm.Min > ulm.Max {
			return retErr(ulmErr(err))
		}

		if len(rest) > 0 && rest[0] == sepseq {
			return ulm, rest[1:], nil
		} else {
			return ulm, nil, nil
		}
	}

	for len(args) > 0 {
		var ulm ULM
		var err error
		ulm, args, err = parseUlm(args)
		if err != nil {
			return nil, err
		}
		ulms = append(ulms, ulm)
	}

	return ulms, nil
}

type systemLaunchManifest struct {
	Name           string                 // configured by the user
	Min, Max       int                    // configured by the user
	ImageId        string                 // located AWS image id
	KeyPair        string                 // configured by the user or generated at run-time
	SecurityGroups []awsec2.SecurityGroup // configured by the user or created at runtime
	Tags           []awsec2.Tag           // configured by oti so that instances can be found later
}
