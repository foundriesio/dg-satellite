// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package storage

import (
	"errors"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type updatesFsHandleWrap struct {
	baseFsHandle
	Apps     UpdatesFsHandle
	Ostree   UpdatesFsHandle
	Tuf      UpdatesFsHandle
	Rollouts RolloutsFsHandle
	Logs     UpdatesFsHandle
}

func (s *updatesFsHandleWrap) init(root string) {
	s.root = root
	s.Apps.root = root
	s.Apps.category = UpdatesAppsDir
	s.Ostree.root = root
	s.Ostree.category = UpdatesOstreeDir
	s.Rollouts.root = root
	s.Rollouts.category = UpdatesRolloutsDir
	s.Tuf.root = root
	s.Tuf.category = UpdatesTufDir
	s.Logs.root = root
	s.Logs.category = UpdatesLogsDir
}

type UpdatesFsHandle struct {
	baseFsHandle
	category string
}

func (s UpdatesFsHandle) FilePath(tag, update, name string) string {
	return filepath.Join(s.root, tag, update, s.category, name)
}

func (s UpdatesFsHandle) ReadFile(tag, update, name string) (string, error) {
	h, _ := s.updateLocalHandle(tag, update, false)
	content, err := h.readFile(name, false)
	if err != nil {
		err = fmt.Errorf("error reading %s file for tag %s update %s: %w", s.category, tag, update, err)
	}
	return content, err
}

func (s UpdatesFsHandle) LatestRootMetaName(tag, update string) (string, error) {
	h, _ := s.updateLocalHandle(tag, update, false)
	files, err := h.matchFiles("", false)
	if err != nil {
		return "", fmt.Errorf("error find latest root metadata: %w", err)
	} else if len(files) == 0 {
		return "", fmt.Errorf("no metadata files found for tag %s update %s", tag, update)
	}
	slices.SortFunc(files, func(a, b string) int {
		aIsRoot := strings.HasSuffix(a, ".root.json")
		bIsRoot := strings.HasSuffix(b, ".root.json")
		if aIsRoot && bIsRoot {
			// Convert both into 0 padded strings to ensure proper lexicographical comparison
			// 13 = makes 1.root.json become 001.root.json to support 999 versions of root.json
			a = fmt.Sprintf("%013s", a)
			b = fmt.Sprintf("%013s", b)
			return strings.Compare(b, a)
		} else if aIsRoot {
			return -1
		} else if bIsRoot {
			return 1
		}
		return 1
	})
	return files[0], nil
}

func (s UpdatesFsHandle) TailFileLines(tag, update, name string, stop DoneChan) iter.Seq2[string, error] {
	h, _ := s.updateLocalHandle(tag, update, false)
	return h.readFileLines(name, false, stop)
}

func (s UpdatesFsHandle) WriteFile(tag, update, name, content string) error {
	if h, err := s.updateLocalHandle(tag, update, true); err != nil {
		return err
	} else if err = h.writeFile(name, content, defaultFileAccess); err != nil {
		return fmt.Errorf("error writing %s file for tag %s update %s: %w", s.category, tag, update, err)
	}
	return nil
}

func (s UpdatesFsHandle) AppendFile(tag, update, name, content string) error {
	if h, err := s.updateLocalHandle(tag, update, true); err != nil {
		return err
	} else if err = h.appendFile(name, content, defaultFileAccess); err != nil {
		return fmt.Errorf("error appending %s file for tag %s update %s: %w", s.category, tag, update, err)
	}
	return nil
}

func (s UpdatesFsHandle) updateLocalHandle(tag, update string, forUpdate bool) (h baseFsHandle, err error) {
	h.root = filepath.Join(s.root, tag, update, s.category)
	if forUpdate {
		if err = h.mkdirs(defaultDirAccess, true); err != nil {
			err = fmt.Errorf("unable to create %s file storage for tag %s update %s: %w", s.category, tag, update, err)
		}
	}
	return
}

type RolloutsFsHandle struct {
	UpdatesFsHandle
}

func (s RolloutsFsHandle) ListUpdates(tag string) (map[string][]string, error) {
	// An assumption is that we will have a limited amount of tags.
	// In this case it is just fine to list all available updates for all tags at once.
	var tagDirs []string
	if len(tag) > 0 {
		tagDirs = []string{tag}
	} else if dirs, err := os.ReadDir(s.root); err == nil {
		for _, d := range dirs {
			if d.IsDir() {
				tagDirs = append(tagDirs, d.Name())
			}
		}
	} else if os.IsNotExist(err) {
		return nil, nil
	} else {
		return nil, err
	}

	res := make(map[string][]string, len(tagDirs))
	for _, tag = range tagDirs {
		if dirs, err := os.ReadDir(filepath.Join(s.root, tag)); err == nil {
			res[tag] = make([]string, 0, len(dirs))
			for _, d := range dirs {
				if d.IsDir() {
					res[tag] = append(res[tag], d.Name())
				}
			}
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}
	return res, nil
}

func (s RolloutsFsHandle) ListFiles(tag, update string) ([]string, error) {
	h, _ := s.updateLocalHandle(tag, update, false)
	return h.matchFiles("", true)
}

func (s RolloutsFsHandle) AppendJournal(content string) error {
	return s.appendFile(rolloutJournalFile+partialFileSuffix, content, defaultFileAccess)
}

func (s RolloutsFsHandle) RolloverJournal() (err error) {
	from := filepath.Join(s.root, rolloutJournalFile+partialFileSuffix)
	to := filepath.Join(s.root, rolloutJournalFile)
	if err = os.Rename(from, to); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// No new writes into a journal since the last rollover - that's just fine.
			err = nil
		}
	}
	return
}

func (s RolloutsFsHandle) ReadJournal() iter.Seq2[string, error] {
	return s.readFileLines(rolloutJournalFile, true, nil)
}
