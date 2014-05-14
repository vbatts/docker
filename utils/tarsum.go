package utils

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"github.com/dotcloud/docker/vendor/src/code.google.com/p/go/src/pkg/archive/tar"
	"hash"
	"io"
	"sort"
	"strconv"
	"strings"
)

/*
A wrapper for an io.Reader that is an archive/tar stream, and provides an
io.Reader that is useable by archive/tar NewReader

Along the way generates a fix-time sha256 hash of the contents of the incoming tar.
*/
type TarSum struct {
	io.Reader
	tarR               *tar.Reader
	tarW               *tar.Writer
	writer             writeCloseFlusher
	bufTar             *bytes.Buffer
	bufWriter          *bytes.Buffer
	h                  hash.Hash
	sums               map[string]string
	currentFile        string
	finished           bool
	first              bool
	DisableCompression bool
}

type writeCloseFlusher interface {
	io.WriteCloser
	Flush() error
}

type nopCloseFlusher struct {
	io.Writer
}

func (n *nopCloseFlusher) Close() error {
	return nil
}

func (n *nopCloseFlusher) Flush() error {
	return nil
}

func (ts *TarSum) encodeHeader(h *tar.Header) error {
	for _, elem := range [][2]string{
		{"name", h.Name},
		{"mode", strconv.Itoa(int(h.Mode))},
		{"uid", strconv.Itoa(h.Uid)},
		{"gid", strconv.Itoa(h.Gid)},
		{"size", strconv.Itoa(int(h.Size))},
		{"mtime", strconv.Itoa(int(h.ModTime.UTC().Unix()))},
		{"typeflag", string([]byte{h.Typeflag})},
		{"linkname", h.Linkname},
		{"uname", h.Uname},
		{"gname", h.Gname},
		{"devmajor", strconv.Itoa(int(h.Devmajor))},
		{"devminor", strconv.Itoa(int(h.Devminor))},
		// {"atime", strconv.Itoa(int(h.AccessTime.UTC().Unix()))},
		// {"ctime", strconv.Itoa(int(h.ChangeTime.UTC().Unix()))},
	} {
		if _, err := ts.h.Write([]byte(elem[0] + elem[1])); err != nil {
			return err
		}
	}
	return nil
}

func (ts *TarSum) initTarSum() error {
	ts.bufTar = bytes.NewBuffer([]byte{})
	ts.bufWriter = bytes.NewBuffer([]byte{})
	ts.tarR = tar.NewReader(ts.Reader)
	ts.tarW = tar.NewWriter(ts.bufTar)
	if !ts.DisableCompression {
		ts.writer = gzip.NewWriter(ts.bufWriter)
	} else {
		ts.writer = &nopCloseFlusher{Writer: ts.bufWriter}
	}
	ts.h = sha256.New()
	ts.h.Reset()
	ts.first = true
	ts.sums = make(map[string]string)
	return nil
}

func (ts *TarSum) Read(buf []byte) (int, error) {
	if ts.writer == nil {
		if err := ts.initTarSum(); err != nil {
			return 0, err
		}
	}

	if ts.finished {
		return ts.bufWriter.Read(buf)
	}
	buf2 := make([]byte, len(buf), cap(buf))

	n, err := ts.tarR.Read(buf2)
	if err != nil && err != io.EOF {
		return n, err
	} else if err != nil {
		if _, err := ts.h.Write(buf2[:n]); err != nil {
			return 0, err
		}
		if !ts.first {
			ts.sums[ts.currentFile] = hex.EncodeToString(ts.h.Sum(nil))
			ts.h.Reset()
		} else {
			ts.first = false
		}

		currentHeader, err := ts.tarR.Next()
		if err != nil && err == io.EOF {
			if err := ts.writer.Close(); err != nil {
				return 0, err
			}
			ts.finished = true
			return n, nil
		} else if err != nil {
			return n, err
		}
		ts.currentFile = strings.TrimSuffix(strings.TrimPrefix(currentHeader.Name, "./"), "/")
		if err := ts.encodeHeader(currentHeader); err != nil {
			return 0, err
		}
		if err := ts.tarW.WriteHeader(currentHeader); err != nil {
			return 0, err
		}
		if _, err := ts.tarW.Write(buf2[:n]); err != nil {
			return 0, err
		}
		ts.tarW.Flush()
		if _, err := io.Copy(ts.writer, ts.bufTar); err != nil {
			return 0, err
		}
		ts.writer.Flush()

		return ts.bufWriter.Read(buf)
	}

	// Filling the hash buffer
	if _, err = ts.h.Write(buf2[:n]); err != nil {
		return 0, err
	}

	// Filling the tar writter
	if _, err = ts.tarW.Write(buf2[:n]); err != nil {
		return 0, err
	}
	ts.tarW.Flush()

	// Filling the output writer
	if _, err = io.Copy(ts.writer, ts.bufTar); err != nil {
		return 0, err
	}
	ts.writer.Flush()

	return ts.bufWriter.Read(buf)
}

func (ts *TarSum) Sum(extra []byte) string {
	var sums []string

	for _, sum := range ts.sums {
		sums = append(sums, sum)
	}
	sort.Strings(sums)
	h := sha256.New()
	if extra != nil {
		h.Write(extra)
	}
	for _, sum := range sums {
		Debugf("-->%s<--", sum)
		h.Write([]byte(sum))
	}
	checksum := "tarsum+sha256:" + hex.EncodeToString(h.Sum(nil))
	Debugf("checksum processed: %s", checksum)
	return checksum
}

func (ts *TarSum) GetSums() map[string]string {
	return ts.sums
}
