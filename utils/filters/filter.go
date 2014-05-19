package filters

import (
	"fmt"
	"sync"
)

// the globally register filters
var rf = NewJobFilters()

// Register a job specific filter, from the globally registered filters
func Register(jobName, filterName string, filter Filter) error {
	return rf.Register(jobName, filterName, filter)
}

// GetFilterSet returns the registered FilterSet for the named job, from the globally registered filters
func GetFilterSet(jobName string) FilterSet {
	return rf.GetFilterSet(jobName)
}

// UnRegister removes the named filter, from the globally registered filters. If the filter was set, then it is returned.
func UnRegister(jobName, filterName string) *Filter {
	return rf.UnRegister(jobName, filterName)
}

// JobFilters is a mechanism for keeping track of the filters, by job
type JobFilters interface {
	Register(jobName, filterName string, filter Filter) error
	GetFilterSet(jobName string) FilterSet
	UnRegister(jobName, filterName string) *Filter
}

// Create an initialized JobFilters
func NewJobFilters() JobFilters {
	return jobFilters{registeredFilters: map[string]FilterSet{}, mutex: sync.Mutex{}}
}

type jobFilters struct {
	registeredFilters map[string]FilterSet
	mutex             sync.Mutex
}

// Register a job specific filter
func (jf jobFilters) Register(jobName, filterName string, filter Filter) error {
	jf.mutex.Lock()
	defer jf.mutex.Unlock()
	if _, ok := jf.registeredFilters[jobName]; !ok {
		jf.registeredFilters[jobName] = FilterSet{}
	}
	if _, ok := jf.registeredFilters[jobName][filterName]; ok {
		return fmt.Errorf("Filter exists for job: %s, filtername: %s", jobName, filterName)
	}

	jf.registeredFilters[jobName][filterName] = filter
	return nil
}

// GetFilterSet returns the registered FilterSet for the named job
func (jf jobFilters) GetFilterSet(jobName string) FilterSet {
	return jf.registeredFilters[jobName]
}

// UnRegister removes the named filter. If the filter was set, then it is returned.
func (jf jobFilters) UnRegister(jobName, filterName string) *Filter {
	jf.mutex.Lock()
	defer jf.mutex.Unlock()
	if _, ok := jf.registeredFilters[jobName]; !ok {
		return nil
	}
	var f *Filter
	if i, ok := jf.registeredFilters[jobName][filterName]; ok {
		delete(jf.registeredFilters[jobName], filterName)
		f = &i
	}
	if len(jf.registeredFilters[jobName]) == 0 {
		delete(jf.registeredFilters, jobName)
	}
	return f
}

// A registered set of filters available (e.g. for images, ps, events)
//   so `docker images` would have a FilterSet. Each key is the name of the filter, and the value is Filter to do the processing
type FilterSet map[string]Filter

// FromArgs takes the filter Args (like parsed from a client), and returns a derived FilterSet that apply to those Args
func (fs FilterSet) FromArgs(args Args) FilterSet {
	ret := FilterSet{}
	for arg, params := range args {
		if _, ok := fs[arg]; ok {
			// we make a copy of the filter, since we're going to attach the args to it
			ret[arg] = NewFilter(fs[arg].Usage, fs[arg].FilterFunc, fs[arg].ValidateArg)
			for i := range params {
				// XXX
				//ret[arg].Values = append(ret[arg].Values, params[i])
				fmt.Println(params[i])
				_ = i
			}
		}
	}
	return ret
}

func NewFilter(usage string, filterFunc FilterFunc, validateArg ValidateArgFunc) Filter {
	return Filter{
		Usage:       usage,
		FilterFunc:  filterFunc,
		ValidateArg: validateArg,
	}
}

// Create an instance for every operand of filter arguments provided
type Filter struct {
	FilterFunc  FilterFunc
	ValidateArg ValidateArgFunc
	Usage       string
	Values      []string // XXX need to not store the args, but link to the filters.Args that matches
}

// The filter function registered. Where value is the operand argument provided,
// and item is from each item in a given list.
// The returned bool is whether the item ought to be filtered out (not included)
type FilterFunc func(value string, item interface{}) bool

// Validate the arg provided for whether it is a valid argument for this filter
type ValidateArgFunc func(arg string) bool
