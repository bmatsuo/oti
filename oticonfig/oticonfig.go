// Copyright 2014, Bryan Matsuo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// oticonfig.go [created: Thu, 20 Mar 2014]

package oticonfig

import (
	"github.com/bmatsuo/go-jsontree"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
)

type C struct {
	AwsKeyPath string // file containing an AwsKey json object
	PackerDir  string // directory containing packer files (w/ .json extension)
	TagPrefix  string // namespace for tag keys used by oti.
}

type OTITag struct{ Key, Value string }

func Read(path string, c *C) error {
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

func (c *C) AwsKey() (*AwsKey, error) {
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

	if k.AccessKey == "" {
		return nil, fmt.Errorf("missing AccessKey")
	}

	if k.SecretKey == "" {
		return nil, fmt.Errorf("missing SecretKey")
	}

	return &k, nil
}

func (c *C) Packer(name string) (*Packer, error) {
	var p Packer
	pp, err := ioutil.ReadFile(filepath.Join(c.PackerDir, name+".json"))
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(pp, &p)
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (c *C) Packers() ([]string, error) {
	ps, err := filepath.Glob(filepath.Join(c.PackerDir, "*.json"))
	if err != nil {
		return nil, err
	}
	for i := range ps {
		ps[i] = strings.TrimSuffix(filepath.Base(ps[i]), ".json")
	}
	return ps, nil
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
