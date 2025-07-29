// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package context

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

var levelMap = map[string]slog.Level{
	"debug":   slog.LevelDebug,
	"info":    slog.LevelInfo,
	"warning": slog.LevelWarn,
	"error":   slog.LevelError,
}

func InitLogger(level string) (*slog.Logger, error) {
	if level == "" {
		level = os.Getenv("LOG_LEVEL")
		if level == "" {
			level = "info"
		}
	}
	logLevel, ok := levelMap[level]
	if !ok {
		var valid []string
		for k := range levelMap {
			valid = append(valid, k)
		}
		return nil, fmt.Errorf("invalid log level: %s; supported: %s", level, strings.Join(valid, ", "))
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	// This sets a default global logger for both slog and legacy log packages.
	slog.SetDefault(logger)
	_ = slog.SetLogLoggerLevel(logLevel)
	return logger, nil
}
