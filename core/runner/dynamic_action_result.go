package runner

import (
	"reflect"

	"github.com/FelixSeptem/baymax/core/types"
)

// Public pause and duplicate results must not expose source-owned state.
func detachedDynamicActionResult(r types.RunResult) types.RunResult {
	r.Warnings = append([]string(nil), r.Warnings...)
	r.ToolCalls = append([]types.ToolCallSummary(nil), r.ToolCalls...)
	for i := range r.ToolCalls {
		r.ToolCalls[i].Error = detachedDynamicActionError(r.ToolCalls[i].Error)
	}
	r.Error = detachedDynamicActionError(r.Error)
	if r.TerminalOutcome != nil {
		t := *r.TerminalOutcome
		if t.PendingAction != nil {
			a := *t.PendingAction
			t.PendingAction = &a
		}
		r.TerminalOutcome = &t
	}
	return r
}

func detachedDynamicActionError(err *types.ClassifiedError) *types.ClassifiedError {
	if err == nil {
		return nil
	}
	x := *err
	if x.Details != nil {
		x.Details = detachedDynamicActionDetail(reflect.ValueOf(x.Details), make(map[dynamicActionDetailVisit]reflect.Value)).Interface().(map[string]any)
	}
	return &x
}

type dynamicActionDetailVisit struct {
	typ     reflect.Type
	pointer uintptr
	length  int
}

// Preserve concrete types in structured details. Tracking visited collections
// also preserves aliases and terminates for cyclic host-provided details.
func detachedDynamicActionDetail(v reflect.Value, seen map[dynamicActionDetailVisit]reflect.Value) reflect.Value {
	var visit dynamicActionDetailVisit
	switch v.Kind() {
	case reflect.Map, reflect.Slice, reflect.Pointer:
		if v.IsNil() {
			return v
		}
		visit = dynamicActionDetailVisit{typ: v.Type(), pointer: uintptr(v.UnsafePointer())}
		if v.Kind() == reflect.Slice {
			visit.length = v.Len()
		}
		if x, ok := seen[visit]; ok {
			return x
		}
	}
	switch v.Kind() {
	case reflect.Interface:
		if !v.IsNil() {
			x := reflect.New(v.Type()).Elem()
			x.Set(detachedDynamicActionDetail(v.Elem(), seen))
			return x
		}
	case reflect.Map:
		if !v.IsNil() {
			x := reflect.MakeMapWithSize(v.Type(), v.Len())
			seen[visit] = x
			iter := v.MapRange()
			for iter.Next() {
				x.SetMapIndex(iter.Key(), detachedDynamicActionDetail(iter.Value(), seen))
			}
			return x
		}
	case reflect.Slice, reflect.Array:
		if v.Kind() == reflect.Slice && v.IsNil() {
			return v
		}
		var x reflect.Value
		if v.Kind() == reflect.Slice {
			x = reflect.MakeSlice(v.Type(), v.Len(), v.Len())
			seen[visit] = x
		} else {
			x = reflect.New(v.Type()).Elem()
		}
		for i := 0; i < v.Len(); i++ {
			x.Index(i).Set(detachedDynamicActionDetail(v.Index(i), seen))
		}
		return x
	case reflect.Pointer:
		x := reflect.New(v.Type().Elem())
		seen[visit] = x
		x.Elem().Set(detachedDynamicActionDetail(v.Elem(), seen))
		return x
	case reflect.Struct:
		x := reflect.New(v.Type()).Elem()
		x.Set(v)
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() {
				x.Field(i).Set(detachedDynamicActionDetail(v.Field(i), seen))
			}
		}
		return x
	}
	return v
}
