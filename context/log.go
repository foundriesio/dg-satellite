// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

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
	logLevel, ok := levelMap[strings.ToLower(level)]
	if !ok {
		var valid []string
		for k := range levelMap {
			valid = append(valid, k)
		}
		return nil, fmt.Errorf("invalid log level: %s; supported: %s", level, strings.Join(valid, ", "))
	}

	opts := &slog.HandlerOptions{Level: logLevel}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, opts))
	// This sets a default global logger for both slog and legacy log packages.
	slog.SetDefault(logger)
	// This tells the log level at which standard log messages should be logged.
	// Let's keep this at Warn, as we do want to eventually clean up all these sneaky logs.
	_ = slog.SetLogLoggerLevel(slog.LevelWarn)
	return logger, nil
}
