package sago

import (
	"database/sql"
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
		return s.returnSelect(
			Copy(reflect.ValueOf(fromCached)),
			nilErr,
		)
	}
	results = s.Select(args)
	if results[1].IsNil() {
		s.Cache.Set(dir, key, Copy(results[0]).Interface())
	}
	return results
}
func (s *SQLExecutor) returnSelect(object reflect.Value, err error) (results []reflect.Value) {
	returnlength := len(s.ReturnTypes)
	switch returnlength {
	case 2:
		return []reflect.Value{
			object,
			reflect.ValueOf(&err).Elem(),
		}
	case 3:
		if err == nil {
			return []reflect.Value{
				object,
				reflect.ValueOf(true),
				reflect.ValueOf(&err).Elem(),
			}
		} else if err == sql.ErrNoRows {
			return []reflect.Value{
				object,
				reflect.ValueOf(false),
				reflect.ValueOf(&nilErr).Elem(),
			}
		} else {
			return []reflect.Value{
				object,
				reflect.ValueOf(false),
				reflect.ValueOf(&err).Elem(),
			}
		}
	}
	panic("select only support any,err or any,exist,err returned")
}
func (s *SQLExecutor) Select(args []reflect.Value) (results []reflect.Value) {
	sqlString, sqlArgs, err := s.executeTpl(args)
	if err != nil {
		return s.returnSelect(
			reflect.Zero(s.ReturnTypes[0]),
			err,
		)
	}

	resultType := s.ReturnTypes[0]
	switch resultType.Kind() {
	case reflect.Slice, reflect.Array:
		listValue := reflect.New(resultType)
		var err error
		err = s.DB.Select(listValue.Interface(), sqlString, sqlArgs...)
		return s.returnSelect(
			listValue.Elem(),
			err,
		)
	case reflect.Ptr:
		oneValue := reflect.New(resultType.Elem())
		var err error
		err = s.DB.Get(oneValue.Interface(), sqlString, sqlArgs...)
		return s.returnSelect(
			oneValue,
			err,
		)

	case reflect.Struct,
		reflect.Bool,
		reflect.String,
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
		return s.returnSelect(
			oneValue.Elem(),
			err,
		)
	default:
		panic("not support such type " + resultType.String())
	}

	return nil
}
