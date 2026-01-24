// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"encoding/json"
	"fmt"
	"io"
)

func (a Api) Get(resource string, result any) error {
	url := a.URL + resource
	resp, err := a.Client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		buf, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("API request failed with status %d and unreadable body", resp.StatusCode)
		}
		rid := resp.Header.Get("X-Request-ID")
		return fmt.Errorf("API request (id=%s) failed with status %d: %s", rid, resp.StatusCode, string(buf))
	}

	return json.NewDecoder(resp.Body).Decode(result)
}
