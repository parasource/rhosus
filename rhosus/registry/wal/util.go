package wal

import (
	"errors"
	"fmt"
	"github.com/parasource/rhosus/rhosus/util/fileutil"
	"github.com/sirupsen/logrus"
	"strings"
)

func isValidSeq(names []string) bool {
	var lastSeq int64
	for _, name := range names {
		curSeq, _, err := parseWALName(name)
		if err != nil {
			logrus.Error("failed to parse wal name")
		}
		if lastSeq != 0 && lastSeq != curSeq-1 {
			return false
		}
		lastSeq = curSeq
	}
	return true
}

func dirExists(dir string) bool {
	names, err := fileutil.ReadDir(dir, fileutil.WithExt(".wal"))
	if err != nil {
		return false
	}
	return len(names) != 0
}

func readWALNames(dirpath string) ([]string, error) {
	names, err := fileutil.ReadDir(dirpath)
	if err != nil {
		return nil, err
	}
	wnames := checkWalNames(names)
	if len(wnames) == 0 {
		return nil, errors.New("wal file not found")
	}
	return wnames, nil
}

func checkWalNames(names []string) []string {
	wnames := make([]string, 0)
	for _, name := range names {
		if _, _, err := parseWALName(name); err != nil {
			// don't complain about left over tmp files
			if !strings.HasSuffix(name, ".tmp") {
				continue
			}
			continue
		}
		wnames = append(wnames, name)
	}
	return wnames
}

func parseWALName(str string) (seq, index int64, err error) {
	if !strings.HasSuffix(str, ".wal") {
		return 0, 0, errors.New("bad wal name")
	}
	_, err = fmt.Sscanf(str, "%016x-%016x.wal", &seq, &index)
	return seq, index, err
}

func walName(seq, index int64) string {
	return fmt.Sprintf("%016x-%016x.wal", seq, index)
}

func searchIndex(names []string, index int64) (int, bool) {
	for i := len(names) - 1; i >= 0; i-- {
		name := names[i]
		_, curIndex, err := parseWALName(name)
		if err != nil {
			return -1, false
		}
		if index >= curIndex {
			return i, true
		}
	}
	return -1, false
}
