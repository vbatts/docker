package filters

import (
	"sort"
	"testing"
)

func TestParseArgs(t *testing.T) {
	// equivalent of `docker ps -f 'created=today' -f 'image.name=ubuntu*' -f 'image.name=*untu'`
	flagArgs := []string{
		"created=today",
		"image.name=ubuntu*",
		"image.name=*untu",
	}
	var (
		args = Args{}
		err  error
	)
	for i := range flagArgs {
		args, err = ParseFlag(flagArgs[i], args)
		if err != nil {
			t.Errorf("failed to parse %s: %s", flagArgs[i], err)
		}
	}
	if len(args["created"]) != 1 {
		t.Logf("args: %#v", args)
		t.Errorf("failed to set this arg")
	}
	if len(args["image.name"]) != 2 {
		t.Logf("args: %#v", args)
		t.Errorf("the args should have collapsed")
	}
}

func TestParam(t *testing.T) {
	a := Args{
		"created":    []string{"today"},
		"image.name": []string{"ubuntu*", "*untu"},
	}

	v, err := ToParam(a)
	if err != nil {
		t.Errorf("failed to marshal the filters: %s", err)
	}
	v1, err := FromParam(v)
	if err != nil {
		t.Errorf("%s", err)
	}
	for key, vals := range v1 {
		if _, ok := a[key]; !ok {
			t.Errorf("could not find key %s in original set", key)
		}
		sort.Strings(vals)
		sort.Strings(a[key])
		if len(vals) != len(a[key]) {
			t.Errorf("value lengths ought to match")
			continue
		}
		for i := range vals {
			if vals[i] != a[key][i] {
				t.Errorf("expected %s, but got %s", a[key][i], vals[i])
			}
		}
	}
}

func TestEmpty(t *testing.T) {
	a := Args{}
	v, err := ToParam(a)
	if err != nil {
		t.Errorf("failed to marshal the filters: %s", err)
	}
	v1, err := FromParam(v)
	if err != nil {
		t.Errorf("%s", err)
	}
	if len(a) != len(v1) {
		t.Errorf("these should both be empty sets")
	}
}

func TestOperatorParse(t *testing.T) {
	for _, filt := range []struct {
		arg   string
		items []string
	}{
		{"created=today", []string{"created", "=", "today"}},
		{"exited!=0", []string{"exited", "!=", "0"}},
		{"created>1d", []string{"created", ">", "1d"}},
		{"baz>=bif", []string{"baz", ">=", "bif"}},
		{"foo<=harfblat", []string{"foo", "<=", "harfblat"}},
	} {
		items, err := SplitByOperators(filt.arg)
		if err != nil {
			t.Errorf("failed to parse [%s]: %s", filt.arg, err)
		}
		if len(items) != len(filt.items) {
			t.Errorf("expected %d items, got %d", len(filt.items), len(items))
		}
		for i := 0; i < 3; i++ {
			if items[i] != filt.items[i] {
				t.Errorf("expected %s items, got %s", filt.items[i], items[i])
			}
		}
	}
}
