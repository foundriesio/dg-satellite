// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package subcommands

import (
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"time"
)

const AnsiClearLine = "\033[2K\r"

func FileProgress(size int64) (p *progress) {
	p = &progress{}
	p.SetTotal(size)
	return
}

func ArchiveProgress(source fileSourcer, sizer func(os.FileInfo) int64) (p *progress, s fileSourcer) {
	// An idea is to spawn a separate file sourcer goroutine which estimates the archive file size,
	// while individual files are being pushed into the channel for archiving.
	p = &progress{}
	fault := make(chan error)
	sink := make(chan *ArchiveEntry, 10000) // each item uses up to 40-200 bytes with reasonable path lengths.
	done := make(chan bool)
	var halt atomic.Bool

	runSizer := func() {
		var (
			entry *ArchiveEntry
			err   error
			total int64
		)
		defer close(done)
		defer close(fault)
		defer close(sink)
		for entry, err = range source {
			if halt.Load() {
				return
			}
			if err != nil {
				break
			}
			if err = entry.EnsureInfo(); err != nil {
				break
			}
			total += sizer(entry.Info)
			sink <- entry
		}
		if err != nil {
			fault <- err
		} else {
			p.SetTotal(total)
		}
	}
	s = func(yield func(*ArchiveEntry, error) bool) {
		defer halt.Store(true)
		go runSizer()
		for sink != nil || fault != nil {
			select {
			case entry, ok := <-sink:
				if !ok {
					sink = nil // sink closed, stop listening
				} else if !yield(entry, nil) {
					return
				}
			case err, ok := <-fault:
				if !ok {
					fault = nil // fault closed, stop listening
				} else {
					_ = yield(nil, err)
					return
				}
			}
		}
		<-done // Make sure that total is set before progress ends in case of success.
	}
	return
}

func TarProgress(source fileSourcer) (*progress, fileSourcer) {
	// Tar adds 512 bytes header for each entry and aligns each file by 512 byte blocks: https://wiki.osdev.org/Tar.
	// This allows us to guesstimate an almost precise tarball size based on its constituents.
	return ArchiveProgress(source, func(info os.FileInfo) (size int64) {
		size = 512 // header is 512 bytes for all entries
		if !info.IsDir() {
			size += (info.Size()/512 + 1) * 512 // file size aligned by 512 bytes
		}
		return size
	})
}

type progress struct {
	count atomic.Int64
	total atomic.Int64
}

func (p *progress) Count() int64 {
	return p.count.Load()
}

func (p *progress) Total() int64 {
	return p.total.Load()
}

func (p *progress) SetTotal(total int64) {
	p.total.Store(total)
}

func (p *progress) StreamWriter(source sourceWriter) sourceWriter {
	return func(writer io.Writer) error {
		w := countingWriter{writer, &p.count}
		return source(w)
	}
}

func (p *progress) StreamReader(reader io.ReadCloser) io.ReadCloser {
	return countingReader{reader, &p.count}
}

func (p *progress) Report(task string, stop, done chan bool) {
	if done != nil {
		defer close(done)
	}
	interval := 500 * time.Millisecond
	ticker := time.NewTicker(interval)
	checkpoint := p.Count()
	for i := int64(1); true; i += 1 {
		select {
		case <-ticker.C:
			count := p.Count()
			total := p.Total()
			rate := int64(float64(count-checkpoint) / interval.Seconds()) // approximate, yet precise enough
			checkpoint = count
			var line string
			if total > 0 {
				pct := float64(count) / float64(total) * 100
				var etaStr string
				if count > 0 {
					eta := (time.Duration((total-count)*i/count) * interval).Truncate(time.Second)
					if eta.Seconds() < 1 {
						eta = time.Second
					}
					etaStr = "ETA " + eta.String()
				}
				line = fmt.Sprintf("%s / %s (%.1f%%) [%s/s] %s",
					formatBytes(count), formatBytes(total), pct, formatBytes(rate), etaStr)
			} else {
				line = fmt.Sprintf("%s [%s/s]", formatBytes(count), formatBytes(rate))
			}
			fmt.Print(AnsiClearLine, task, line) // Always prints on the same line
		case complete := <-stop:
			if complete {
				// If task is complete, percent done is always 100%.
				count := formatBytes(p.Count())
				fmt.Print(AnsiClearLine, task, count, "/", count, " (100%%)")
			}
			fmt.Println() // We no longer need an AnsiClearLine, end progress with a newline.
			return
		}
	}
}

type countingWriter struct {
	writer io.Writer
	count  *atomic.Int64
}

func (w countingWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	w.count.Add(int64(n))
	return n, err
}

func (w countingWriter) Close() error {
	return nil
}

type countingReader struct {
	reader io.Reader
	count  *atomic.Int64
}

func (r countingReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.count.Add(int64(n))
	return n, err
}

func (r countingReader) Close() error {
	return nil
}

func formatBytes(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.2f GB", float64(b)/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.2f MB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.2f KB", float64(b)/(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
