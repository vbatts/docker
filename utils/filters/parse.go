package filters

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// the key/[]value set of filters passed from the client
// where in `--filter 'exited!=0'`, 'exited' is the key, and '!=0' is added to the list of values
type Args map[string][]string

// Parse the argument to the filter flag. Like
//
//   `docker ps -f 'created=today' -f 'image.name=ubuntu*'`
//
// If prev map is provided, then it is appended to, and returned. By default a new
// map is created.
func ParseFlag(arg string, prev Args) (Args, error) {
	var filters Args = prev
	if prev == nil {
		filters = Args{}
	}
	if len(arg) == 0 {
		return filters, nil
	}

	if !strings.ContainsAny(arg, Operators) {
		//return filters, ErrorBadFormat
		return filters, fmt.Errorf("did not find operators [%q] in [%q]", Operators, arg)
	}

	items, err := SplitByOperators(arg)
	if err != nil {
		return filters, err
	}
	filters[items[0]] = append(filters[items[0]], strings.Join(items[1:], ""))

	return filters, nil
}

var ErrorBadFormat = errors.New("bad format of filter (e.g. expected name=value)")

// packs the Args into an string for easy transport from client to server
func ToParam(a Args) (string, error) {
	// this way we don't URL encode {}, just empty space
	if len(a) == 0 {
		return "", nil
	}

	buf, err := json.Marshal(a)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

// unpacks the filter Args
func FromParam(p string) (Args, error) {
	args := Args{}
	if len(p) == 0 {
		return args, nil
	}
	err := json.Unmarshal([]byte(p), &args)
	if err != nil {
		return nil, err
	}
	return args, nil
}

func SplitByOperators(str string) ([]string, error) {
	argReader := bytes.NewBufferString(str)
	s := bufio.NewScanner(argReader)
	s.Split(ScanOperators)

	items := []string{}
	for s.Scan() {
		items = append(items, s.Text())
	}
	return items, nil
}

const Operators = "<>=!"

// a scanner splitter to get words and operators
func ScanOperators(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	i := bytes.IndexAny(data, Operators)
	if i == -1 {
		return bufio.ScanWords(data, atEOF)
	}

	// ensure the beginning is not a space or newline
	start := 0
	for width := 0; start < len(data); start += width {
		var r rune
		r, width = utf8.DecodeRune(data[start:])
		if !isSpace(r) {
			break
		}
	}

	if i != 0 {
		return i, data[start:i], nil
	}
	for width, j := 0, 0; j < len(data); j += width {
		var r rune
		r, width = utf8.DecodeRune(data[j:])
		if !isOperator(r) {
			return j, data[0:j], nil
		}
	}
	// Request more data.
	return 0, nil, nil

}
func isSpace(r rune) bool {
	return r == '\n' || r == '\r' || r == ' '
}
func isOperator(r rune) bool {
	for _, c := range Operators {
		if r == c {
			return true
		}
	}
	return false
}
