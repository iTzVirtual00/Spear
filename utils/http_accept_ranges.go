package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type RangeError struct {
	s string
}

func (err RangeError) Error() string {
	return err.s
}

type Range struct {
	Start uint64
	End   uint64
	Size  uint64
}

func (r Range) String() string {
	return fmt.Sprintf("Range{%d-%d, %d}", r.Start, r.End, r.Size)
}

// [start, end]
func NewRange(start uint64, end uint64) Range {
	return Range{Start: start, End: end, Size: end - start + 1}
}

// umaxR max Size value (exclusive), usually file.Size() is ok
// remember that 0-2 serves the first 3 bytes, so umaxR should be 3
func ParseRangeHeader(header string, umaxR uint64, maxChunkSize uint64) ([]Range, error) {
	var ranges []Range
	cleanHeader := strings.ReplaceAll(header, " ", "")
	checkRegex := regexp.MustCompile(`[^,\-\d]`)
	if checkRegex.MatchString(cleanHeader) {
		return nil, &RangeError{fmt.Sprintf("Invalid header: %s", cleanHeader)}
	}

	matches := strings.Split(cleanHeader, ",")

	for _, r := range matches {
		if strings.Count(r, "-") != 1 {
			return nil, &RangeError{fmt.Sprintf("Invalid range: %s", r)}
		}
		if strings.HasPrefix(r, "-") { // starts with
			// parses -b
			i, err := strconv.Atoi(r[1:])
			num := uint64(i)
			if err != nil || num >= umaxR {
				return nil, &RangeError{fmt.Sprintf("Invalid range: %s", r)}
			}
			ranges = append(ranges, NewRange(umaxR-num-1, umaxR-1))
		} else if strings.HasSuffix(r, "-") { // ends with
			// parses a-
			i, err := strconv.Atoi(r[:len(r)-1])
			num := uint64(i)
			if err != nil || num >= umaxR {
				return nil, &RangeError{fmt.Sprintf("Invalid range: %s", r)}
			}
			ranges = append(ranges, NewRange(num, umaxR-1))
		} else if strings.Contains(r, "-") {
			// parses a-b
			slices := strings.Split(r, "-")
			if len(slices) != 2 {
				return nil, &RangeError{fmt.Sprintf("Invalid range: %s", r)}
			}
			a, err := strconv.Atoi(slices[0])
			if err != nil {
				return nil, &RangeError{fmt.Sprintf("Invalid range: %s", r)}
			}
			b, err := strconv.Atoi(slices[1])
			if err != nil || b >= int(umaxR) || a >= b {
				return nil, &RangeError{fmt.Sprintf("Invalid range: %s", r)}
			}
			ranges = append(ranges, NewRange(uint64(a), uint64(b)))
		} else {
			// this should be an impossible state
			return nil, &RangeError{fmt.Sprintf("Invalid range: %s", r)}
		}
		if len(ranges) > 1 {
			added := ranges[len(ranges)-1]
			lastValid := ranges[len(ranges)-2]
			if lastValid.End > added.Start {
				return nil, &RangeError{fmt.Sprintf("Last added range %s overlaps with %s", added, lastValid)}
			}
		}

	}

	return ranges, nil
}
