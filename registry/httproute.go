package registry

import (
	"bufio"
	"io"
	"strings"
)

type HTTPRoute string

// return the formatted route, replacing the values[key] in the named pattern of the HTTPRoute
// e.g HTTPRoute("/path/{foo}/here").Format(map[string]string{"foo":"to"})
//  would return "/path/to/here"
func (hr HTTPRoute) Format(values map[string]string) string {
	str := string(hr)
	// if this route has no fields, then return early
	if strings.Index(str, "{") == -1 {
		return str
	}
	retStr := ""
	rdr := bufio.NewReader(strings.NewReader(str))
	for {
		buf, err := rdr.ReadString('{')
		if (err != nil && err == io.EOF) || strings.Index(buf, "{") == -1 {
			retStr = retStr + buf
			break
		}
		retStr = retStr + buf[:len(buf)-1]

		buf, err = rdr.ReadString('}')
		if err != nil && err == io.EOF {
			break
		}

		key := strings.Split(buf[:len(buf)-1], ":")[0]
		if _, ok := values[key]; !ok {
			retStr = retStr + "{" + key + "}"
		} else {
			retStr = retStr + values[key]
		}
	}
	return retStr
}
func (hr HTTPRoute) Keys() []string {
	if strings.Index(string(hr), "{") == -1 {
		return nil
	}
	keys := []string{}
	rdr := bufio.NewReader(strings.NewReader(string(hr)))
	for {
		buf, err := rdr.ReadString('{')
		if (err != nil && err == io.EOF) || strings.Index(buf, "{") == -1 {
			break
		}
		buf, err = rdr.ReadString('}')
		if err != nil && err == io.EOF {
			break
		}
		keys = append(keys, strings.Split(buf[:len(buf)-1], ":")[0])
	}
	return keys
}
