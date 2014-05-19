package filters

import (
	"errors"
	"github.com/dotcloud/docker/pkg/beam/data"
	"strings"
)

var ArgPrefix = "filter."

type Args map[string][]string

/*
Parse the argument to the filter flag. Like

  `docker ps -f 'created=today' -f 'image.name=ubuntu*'`

If prev map is provided, then it is appended to, and returned. By default a new
map is created.
*/
func ParseFlag(arg string, prev Args) (Args, error) {
	var filters Args = prev
	if prev == nil {
		filters = Args{}
	}
	if len(arg) == 0 {
		return filters, nil
	}

	if !strings.Contains(arg, "=") {
		return filters, ErrorBadFormat
	}

	f := strings.SplitN(arg, "=", 2)
	filters[f[0]] = append(filters[f[0]], f[1])

	return filters, nil
}

var ErrorBadFormat = errors.New("bad format of filter (expected name=value)")

/*
merges the Args into an existing url.Values and returns that product

This appends a "filter." prefix to the keys, to not conflict with existing values
*/
func ToParam(a Args) string {
	return data.Encode(a)
}

/*
from a url.Values, extract the filter Args

This is based on a "filter." prefix of the keys
*/
func FromParam(p string) (Args, error) {
	return data.Decode(p)
}
