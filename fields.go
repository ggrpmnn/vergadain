package main

import (
	"strings"

	sj "github.com/bitly/go-simplejson"
)

// FieldData structs contain data on an individual field
type FieldData struct {
	Name   string
	ID     string
	Values []FieldValue
}

// FieldValue structs contain data on a value for a particular field
type FieldValue struct {
	Value string
	ID    string
}

// get the issuetypes within a particular project
func getTypes(p *sj.Json, vals map[string]FieldData) {
	for i := range p.Get("issuetypes").MustArray() {
		it := p.Get("issuetypes").GetIndex(i)
		getFields(it, vals)
	}
}

// get the fields within a particular issuetype
func getFields(it *sj.Json, vals map[string]FieldData) {
	for k := range it.Get("fields").MustMap() {
		if strings.HasPrefix(k, "customfield_") {
			field := it.Get("fields").Get(k)
			// get the allowed values for this field
			name := field.Get("name").MustString()
			if _, ok := vals[name]; !ok {
				values := getAllowedValues(field)
				fd := FieldData{Name: name, ID: k, Values: values}
				vals[name] = fd
			}
		}
	}
}

// get the allowed values for each field
func getAllowedValues(field *sj.Json) []FieldValue {
	fv := make([]FieldValue, 0)
	for i := range field.Get("allowedValues").MustArray() {
		value := field.Get("allowedValues").GetIndex(i)
		fv = append(fv, FieldValue{Value: value.Get("value").MustString(), ID: value.Get("id").MustString()})
	}
	return fv
}
