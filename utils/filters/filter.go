package filters

/*
A registered set of filters available (e.g. for images, ps, events)
*/
type FilterSet map[string]FilterFunc

/*
The filter function registered. Where value is the operand argument provided,
and item is from each item in a given list.
*/
type FilterFunc func(value string, item interface{}) bool

/*
Create an instance for every operand of filter arguments provided
*/
type Filter struct {
	Func  FilterFunc
	Value string
}
