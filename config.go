// Copyright 2014, Bryan Matsuo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// config.go [created: Thu, 20 Mar 2014]

package main

import (
	"github.com/bmatsuo/go-jsontree"

	"encoding/json"
	"io/ioutil"
)

type OTIConfig struct {
	AwsKeyPath string // file containing an AwsKey json object
	PackerDir  string // directory containing packer files (w/ .json extension)
	Identity   OTITag // tag identifying this oti install (e.g. "eric's pc")
	Agent      OTITag // tag to put on all resources managed by oti
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

func (c *OTIConfig) Packer(name string) (*Packer, error) {
	var p Packer
	pp, err := ioutil.ReadFile(c.AwsKeyPath)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(pp, &p)
	if err != nil {
		return nil, err
	}

	return &p, nil
}

type AwsKey struct {
	AccessKey string
	SecretKey string
}

type Packer struct {
	Vars         *jsontree.JsonTree
	Builders     []*jsontree.JsonTree
	Provisioners []*jsontree.JsonTree
}
