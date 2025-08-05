// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package dg

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/foundriesio/dg-satellite/storage"
)

func (d *DgDevice) TestCreate(targetName string, testName, testId string) error {
	t := storage.TargetTest{
		Uuid:       testId,
		Name:       testName,
		TargetName: targetName,
		Status:     "RUNNING",
		CreatedOn:  time.Now().UTC().Unix(),
	}
	testBytes, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("unexpected error marshaling test json: %w", err)
	}
	return d.storage.fs.WriteFile(d.Uuid, fmt.Sprintf("%s%s", storage.TestsPrefix, testId), testBytes)
}

func (d *DgDevice) TestComplete(testId, status, details string, results []storage.TargetTestResult) error {
	if status != "PASSED" && status != "FAILED" {
		return fmt.Errorf("invalid test status: %s. Must be PASSED or FAILED", status)
	}

	var t storage.TargetTest
	if err := d.storage.fs.ReadAsJson(d.Uuid, fmt.Sprintf("%s%s", storage.TestsPrefix, testId), &t); err != nil {
		return fmt.Errorf("failed to read test data for %s: %w", testId, err)
	}

	for _, res := range results {
		if res.Status != "PASSED" && res.Status != "FAILED" && res.Status != "SKIPPED" {
			return fmt.Errorf("invalid test-result status: %s. Must be PASSED, FAILED, or SKIPPED", res.Status)
		}
	}

	ts := time.Now().UTC().Unix()
	t.Status = status
	t.Details = details
	t.CompletedOn = &ts
	t.Results = results

	testBytes, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("unexpected error marshaling test json: %w", err)
	}

	if err := d.storage.fs.WriteFile(d.Uuid, fmt.Sprintf("%s%s", storage.TestsPrefix, testId), testBytes); err != nil {
		return fmt.Errorf("failed to save completed test data for %s: %w", testId, err)
	}
	return nil
}

func (d *DgDevice) TestStoreArtifact(testId, path string, body io.Reader) error {
	name := fmt.Sprintf("%s-%s_%s", storage.TestArtifactsPrefix, testId, path)
	if strings.ContainsRune(name, '/') || strings.Contains(name, "..") {
		return fmt.Errorf("invalid artifact name: %s", name)
	}
	return d.storage.fs.WriteFileStream(d.Uuid, name, body)
}
