package planbridge

import (
	"bytes"
	"encoding"
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"
)

var defaultLimits = Limits{65536, 128, 1024, 1024, 8, 65536}

func resolveLimits(l Limits) (Limits, error) {
	v := []*int{&l.DescriptorBytes, &l.StructuralItems, &l.Dependencies, &l.ReferenceBytes, &l.NestingDepth, &l.BindingBytes}
	d := []int{defaultLimits.DescriptorBytes, defaultLimits.StructuralItems, defaultLimits.Dependencies, defaultLimits.ReferenceBytes, defaultLimits.NestingDepth, defaultLimits.BindingBytes}
	for i, p := range v {
		if *p < 0 || *p > d[i] {
			return Limits{}, fail(InvalidPlan, "limits", nil)
		}
		if *p == 0 {
			*p = d[i]
		}
	}
	return l, nil
}

// checkSnapshot charges scalar bytes and a byte per container entry before
// encoding or cloning. Encoding is therefore bounded even for escaping-heavy
// strings. The final exact encoded byte limit is checked independently.
func checkSnapshot(v reflect.Value, remaining *int, depth, maxDepth int, jsonLike bool) error {
	if !v.IsValid() {
		return nil
	}
	if v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil
		}
		return checkSnapshot(v.Elem(), remaining, depth, maxDepth, true)
	}
	if jsonLike && (v.Type().Implements(reflect.TypeOf((*json.Marshaler)(nil)).Elem()) || v.Type().Implements(reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()) || reflect.PointerTo(v.Type()).Implements(reflect.TypeOf((*json.Marshaler)(nil)).Elem()) || reflect.PointerTo(v.Type()).Implements(reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem())) {
		return fail(InvalidPlan, "snapshot.value", nil)
	}
	if v.Kind() == reflect.Pointer {
		if jsonLike {
			return fail(InvalidPlan, "binding.value", nil)
		}
		if v.IsNil() {
			return nil
		}
		return checkSnapshot(v.Elem(), remaining, depth, maxDepth, false)
	}
	charge := func(n int) error {
		if n > *remaining {
			return fail(LimitExceeded, "snapshot.bytes", nil)
		}
		*remaining -= n
		return nil
	}
	if err := charge(1); err != nil {
		return err
	}
	switch v.Kind() {
	case reflect.String:
		if !utf8.ValidString(v.String()) {
			return fail(InvalidPlan, "snapshot.value", nil)
		}
		return charge(v.Len())
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return nil
	case reflect.Float32, reflect.Float64:
		if math.IsNaN(v.Float()) || math.IsInf(v.Float(), 0) {
			return fail(InvalidPlan, "snapshot.value", nil)
		}
		return nil
	case reflect.Map, reflect.Slice, reflect.Array:
		if depth >= maxDepth {
			return fail(LimitExceeded, "snapshot.depth", nil)
		}
		if v.Len() > *remaining {
			return fail(LimitExceeded, "snapshot.bytes", nil)
		}
		if v.Kind() == reflect.Map {
			if v.Type().Key().Kind() != reflect.String {
				return fail(InvalidPlan, "snapshot.value", nil)
			}
			it := v.MapRange()
			for it.Next() {
				if err := checkSnapshot(it.Key(), remaining, depth, maxDepth, true); err != nil {
					return err
				}
				if err := checkSnapshot(it.Value(), remaining, depth+1, maxDepth, true); err != nil {
					return err
				}
			}
		} else {
			for i := 0; i < v.Len(); i++ {
				if err := checkSnapshot(v.Index(i), remaining, depth+1, maxDepth, jsonLike); err != nil {
					return err
				}
			}
		}
	case reflect.Struct:
		if jsonLike {
			return fail(InvalidPlan, "snapshot.value", nil)
		}
		if v.Type() == reflect.TypeOf(time.Time{}) {
			return charge(32)
		}
		pkg := v.Type().PkgPath()
		if !strings.HasPrefix(pkg, "github.com/FelixSeptem/baymax/") {
			return fail(InvalidPlan, "snapshot.value", nil)
		}
		for i := 0; i < v.NumField(); i++ {
			f := v.Type().Field(i)
			if f.PkgPath != "" || f.Tag.Get("json") == "-" {
				continue
			}
			if err := checkSnapshot(v.Field(i), remaining, depth, maxDepth, false); err != nil {
				return err
			}
		}
	default:
		return fail(InvalidPlan, "snapshot.value", nil)
	}
	return nil
}
func cloneValue(v reflect.Value) reflect.Value {
	switch v.Kind() {
	case reflect.Interface:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		out := reflect.New(v.Type()).Elem()
		out.Set(cloneValue(v.Elem()))
		return out
	case reflect.Pointer:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		out := reflect.New(v.Type().Elem())
		out.Elem().Set(cloneValue(v.Elem()))
		return out
	case reflect.Map:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		out := reflect.MakeMapWithSize(v.Type(), v.Len())
		it := v.MapRange()
		for it.Next() {
			out.SetMapIndex(it.Key(), cloneValue(it.Value()))
		}
		return out
	case reflect.Slice:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		out := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
		for i := 0; i < v.Len(); i++ {
			out.Index(i).Set(cloneValue(v.Index(i)))
		}
		return out
	case reflect.Array:
		out := reflect.New(v.Type()).Elem()
		for i := 0; i < v.Len(); i++ {
			out.Index(i).Set(cloneValue(v.Index(i)))
		}
		return out
	case reflect.Struct:
		if v.Type() == reflect.TypeOf(time.Time{}) {
			return v
		}
		out := reflect.New(v.Type()).Elem()
		for i := 0; i < v.NumField(); i++ {
			field := v.Type().Field(i)
			if field.PkgPath != "" {
				continue
			}
			if field.Tag.Get("json") == "-" {
				out.Field(i).Set(v.Field(i))
			} else {
				out.Field(i).Set(cloneValue(v.Field(i)))
			}
		}
		return out
	default:
		return v
	}
}
func snapshot[T any](input T, bytes, depth int) (T, error) {
	var zero T
	remaining := bytes
	if err := checkSnapshot(reflect.ValueOf(input), &remaining, 0, depth, false); err != nil {
		return zero, err
	}
	raw, err := json.Marshal(input)
	if err != nil {
		return zero, fail(InvalidPlan, "snapshot.value", nil)
	}
	if len(raw) > bytes {
		return zero, fail(LimitExceeded, "snapshot.bytes", nil)
	}
	return cloneValue(reflect.ValueOf(input)).Interface().(T), nil
}
func encoded(v any) ([]byte, error) {
	raw, e := json.Marshal(v)
	if e != nil {
		return nil, fail(InvalidPlan, "facts", nil)
	}
	return raw, nil
}
func asFacts(v any) map[string]any {
	raw, _ := json.Marshal(v)
	var result map[string]any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	_ = d.Decode(&result)
	return result
}
