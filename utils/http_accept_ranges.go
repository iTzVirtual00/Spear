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
	start uint
	end   uint
}

func (r Range) String() string {
	return fmt.Sprintf("Range{%d-%d}", r.start, r.end)
}

func ParseRange(header string, umaxR uint) ([]Range, error) {
	ranges := []Range{}
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
			num := uint(i)
			if err != nil || num >= umaxR {
				return nil, &RangeError{fmt.Sprintf("Invalid range: %s", r)}
			}
			ranges = append(ranges, Range{
				umaxR - num - 1,
				umaxR - 1,
			})
		} else if strings.HasSuffix(r, "-") { // ends with
			// parses a-
			i, err := strconv.Atoi(r[:len(r)-1])
			num := uint(i)
			if err != nil || num >= umaxR {
				return nil, &RangeError{fmt.Sprintf("Invalid range: %s", r)}
			}
			ranges = append(ranges, Range{
				num,
				umaxR - 1,
			})
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
			ranges = append(ranges, Range{
				uint(a),
				uint(b),
			})
		} else {
			// this should be an impossible state
			return nil, &RangeError{fmt.Sprintf("Invalid range: %s", r)}
		}
		if len(ranges) > 1 {
			added := ranges[len(ranges)-1]
			lastValid := ranges[len(ranges)-2]
			if lastValid.end > added.start {
				return nil, &RangeError{fmt.Sprintf("Last added range %s overlaps with %s", added, lastValid)}
			}
		}

	}

	return ranges, nil
}
