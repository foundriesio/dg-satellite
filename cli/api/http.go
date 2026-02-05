// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func (a Api) Get(resource string, result any) error {
	url := a.URL + resource
	resp, err := a.Client.Get(url)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("warning: failed to close response body: %v\n", err)
		}
	}()

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

func (a Api) Put(resource string, body any) ([]byte, error) {
	url := a.URL + resource

	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("warning: failed to close response body: %v\n", err)
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		buf, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("API request failed with status %d and unreadable body", resp.StatusCode)
		}
		rid := resp.Header.Get("X-Request-ID")
		return nil, fmt.Errorf("API request (id=%s) failed with status %d: %s", rid, resp.StatusCode, string(buf))
	}

	return io.ReadAll(resp.Body)
}

func (a Api) GetStream(resource string) (io.ReadCloser, error) {
	url := a.URL + resource
	resp, err := a.Client.Get(url)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		defer func() {
			if err := resp.Body.Close(); err != nil {
				fmt.Printf("warning: failed to close response body: %v\n", err)
			}
		}()
		buf, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("API request failed with status %d and unreadable body", resp.StatusCode)
		}
		rid := resp.Header.Get("X-Request-ID")
		return nil, fmt.Errorf("API request (id=%s) failed with status %d: %s", rid, resp.StatusCode, string(buf))
	}

	// Return the response without closing the body - caller must close it
	return resp.Body, nil
}
