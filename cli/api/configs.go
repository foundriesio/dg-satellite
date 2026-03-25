// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"io"
)

type ConfigsApi struct {
	api *Api
}

func (a *Api) Configs() ConfigsApi {
	return ConfigsApi{api: a}
}

func (a ConfigsApi) Upload(r io.Reader, opts ...HttpOption) error {
	_, err := a.api.Put("/v1/configs", r, opts...)
	return err
}
