package stream

import (
	"errors"
	"path"
	"strconv"
	"strings"
)

var (
	ErrInvalidRange = errors.New("invalid range")
)

func fileWithID(name, id string) string {
	ext := path.Ext(name)
	return strings.TrimSuffix(name, ext) + "." + id + ext
}

func fileIDFromName(name string) (string, error) {
	parts := strings.Split(name, ".")
	if len(parts) <= 2 {
		return "", errors.New("stream: no ID in name")
	}

	return parts[len(parts)-2], nil
}

func folderWithID(name, id string) string {
	return name + " [" + id + "]"
}

func folderIDFromName(name string) (string, string, error) {
	start := strings.LastIndex(name, "[")
	if start == -1 {
		return "", "", errors.New("stream: no id found")
	}

	start += len("[")
	end := strings.LastIndex(name, "]")
	if end == -1 {
		return "", "", errors.New("stream: no id found")
	}

	return name[:start-2], name[start:end], nil
}

func parseRange(header string, contentLength uint64) (uint64, uint64, error) {
	if header == "" {
		return 0, 0, ErrInvalidRange
	}

	const prefix = "bytes="
	if !strings.HasPrefix(header, prefix) {
		return 0, 0, ErrInvalidRange
	}

	ra := strings.TrimSpace(header[len(prefix):])
	if ra == "" {
		return 0, 0, ErrInvalidRange
	}

	i := strings.Index(ra, "-")
	if i < 0 {
		return 0, 0, ErrInvalidRange
	}

	start, end := strings.TrimSpace(ra[:i]), strings.TrimSpace(ra[i+1:])

	// if no start is given, then end is relative to the end of the file
	if start == "" {
		offset, err := strconv.ParseUint(end, 10, 64)
		if err != nil {
			return 0, 0, ErrInvalidRange
		}

		startPos := contentLength - offset
		if startPos < 0 {
			startPos = 0
		}

		return startPos, contentLength - 1, nil
	}

	startPos, err := strconv.ParseUint(start, 10, 64)
	if err != nil {
		return 0, 0, ErrInvalidRange
	}

	if startPos >= contentLength {
		return 0, 0, ErrInvalidRange
	}

	if end == "" {
		return startPos, contentLength - 1, nil
	}

	endPos, err := strconv.ParseUint(end, 10, 64)
	if err != nil {
		return 0, 0, ErrInvalidRange
	}

	if endPos >= contentLength {
		endPos = contentLength - 1
	}

	if endPos < startPos {
		return 0, 0, ErrInvalidRange
	}

	return startPos, endPos, nil
}
