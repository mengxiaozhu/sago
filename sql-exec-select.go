package sago

import (
	"fmt"
	"reflect"
)

var nilErr error

func (s *SQLExecutor) SelectCache(args []reflect.Value) (results []reflect.Value) {
	keys := []interface{}{}
	for _, v := range args {
		keys = append(keys, v.Interface())
	}
	dir := s.daoName + "." + s.Fn.Name
	key := fmt.Sprint(keys)
	fromCached, ok := s.Cache.Get(dir, key)
	if ok {
		return []reflect.Value{
			Copy(reflect.ValueOf(fromCached)),
			reflect.ValueOf(&nilErr).Elem(),
		}
	}
	results = s.Select(args)
	if results[1].IsNil() {
		s.Cache.Set(dir, key, Copy(results[0]).Interface())
	}
	return results
}
func (s *SQLExecutor) Select(args []reflect.Value) (results []reflect.Value) {
	sqlString, sqlArgs, err := s.executeTpl(args)
	if err != nil {
		return []reflect.Value{
			reflect.Zero(s.ReturnTypes[0]),
			reflect.ValueOf(err),
		}
	}

	resultType := s.ReturnTypes[0]
	switch resultType.Kind() {
	case reflect.Slice, reflect.Array:

		listValue := reflect.New(resultType)
		var err error
		err = s.DB.Select(listValue.Interface(), sqlString, sqlArgs...)
		return []reflect.Value{
			listValue.Elem(),
			reflect.ValueOf(&err).Elem(),
		}
	case reflect.Ptr:
		oneValue := reflect.New(resultType.Elem())
		var err error
		err = s.DB.Get(oneValue.Interface(), sqlString, sqlArgs...)
		return []reflect.Value{
			oneValue.Elem(),
			reflect.ValueOf(&err).Elem(),
		}

	case reflect.Struct,
		reflect.Bool,
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Float32,
		reflect.Float64:
		oneValue := reflect.New(resultType)
		err := s.DB.Get(oneValue.Interface(), sqlString, sqlArgs...)
		return []reflect.Value{
			oneValue.Elem(),
			reflect.ValueOf(&err).Elem(),
		}
	default:
		panic("not support such type " + resultType.String())
	}

	return nil
}
