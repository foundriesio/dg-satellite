// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/foundriesio/dg-satellite/storage"
)

func (d *Device) Tests() ([]storage.TargetTest, error) {
	var tests []storage.TargetTest
	files, err := d.storage.fs.Devices.ListFiles(d.Uuid, storage.TestsPrefix, true)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return []storage.TargetTest{}, nil
	} else if err != nil {
		return nil, err
	}

	for _, file := range files {
		testData, err := d.storage.fs.Devices.ReadFile(d.Uuid, file)
		if err != nil {
			return nil, err
		}
		var test storage.TargetTest
		if err := json.Unmarshal([]byte(testData), &test); err != nil {
			return nil, fmt.Errorf("unexpected error unmarshalling %s: %w", file, err)
		}
		tests = append(tests, test)
	}
	return tests, nil
}

func (d *Device) Test(testId string) (*storage.TargetTest, error) {
	name := fmt.Sprintf("%s%s", storage.TestsPrefix, testId)
	var test storage.TargetTest
	if err := d.storage.fs.Devices.ReadAsJson(d.Uuid, name, &test); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("error reading test %s: %w", testId, err)
	}

	prefix := fmt.Sprintf("%s-%s_", storage.TestArtifactsPrefix, testId)
	files, err := d.storage.fs.Devices.ListFiles(d.Uuid, prefix, true)
	if err != nil {
		return nil, fmt.Errorf("error listing artifacts for test %s: %w", testId, err)
	}
	test.Artifacts = files
	return &test, nil
}

func (d *Device) TestArtifactPath(testId, name string) string {
	name = fmt.Sprintf("%s-%s_%s", storage.TestArtifactsPrefix, testId, name)
	return d.storage.fs.Devices.FilePath(d.Uuid, name)
}
