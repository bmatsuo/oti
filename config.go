// Copyright 2014, Bryan Matsuo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// config.go [created: Thu, 20 Mar 2014]

package main

import (
	"encoding/json"
	"io/ioutil"
)

type OTIConfig struct {
	AwsKeyPath  string
	ResourceTag OTITag
}

type OTITag struct{ Key, Value string }

func readConfig(path string, c *OTIConfig) error {
	configp, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	err = json.Unmarshal(configp, c)
	if err != nil {
		return err
	}
	return nil
}

func (c *OTIConfig) AwsKey() (*AwsKey, error) {
	// TODO stat and warn if permissions are not strict

	keyp, err := ioutil.ReadFile(c.AwsKeyPath)
	if err != nil {
		return nil, err
	}

	var k AwsKey
	err = json.Unmarshal(keyp, &k)
	if err != nil {
		return nil, err
	}

	return &k, nil
}

type AwsKey struct {
	AccessKey string
	SecretKey string
}
