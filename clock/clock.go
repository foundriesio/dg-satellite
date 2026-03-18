// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package clock

import (
	"time"
)

type now func() time.Time

// Golang has a hard time altering a wall clock, including via the libfaketime.
// This abstraction allows to easily amend the wall clock in tests.
var Now now = time.Now
