package sago

import (
	"database/sql"
	"fmt"
	"reflect"
)

var nilErr error

func (e *SQLExecutor) SelectCache(args []reflect.Value) (results []reflect.Value) {
	var keys []interface{}
	for _, v := range args {
		keys = append(keys, v.Interface())
	}
	dir := e.daoName + "." + e.Fn.Name
	key := fmt.Sprint(keys)
	fromCached, ok := e.Cache.Get(dir, key)
	if ok {
		return e.returnSelect(
			clone(reflect.ValueOf(fromCached)),
			nilErr,
		)
	}
	results = e.Select(args)
	if results[1].IsNil() {
		e.Cache.Set(dir, key, clone(results[0]).Interface())
	}
	return results
}

func (e *SQLExecutor) returnSelect(object reflect.Value, err error) (results []reflect.Value) {
	outNum := len(e.ReturnTypes)
	switch outNum {
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

func (e *SQLExecutor) Select(args []reflect.Value) (results []reflect.Value) {
	sqlString, sqlArgs, err := e.executeTpl(args)
	if err != nil {
		return e.returnSelect(
			reflect.Zero(e.ReturnTypes[0]),
			err,
		)
	}

	resultType := e.ReturnTypes[0]
	switch resultType.Kind() {
	case reflect.Slice, reflect.Array:
		listValue := reflect.New(resultType)
		var err error
		err = e.DB.Select(listValue.Interface(), sqlString, sqlArgs...)
		return e.returnSelect(
			listValue.Elem(),
			err,
		)
	case reflect.Ptr:
		oneValue := reflect.New(resultType.Elem())
		var err error
		err = e.DB.Get(oneValue.Interface(), sqlString, sqlArgs...)
		return e.returnSelect(
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
		err := e.DB.Get(oneValue.Interface(), sqlString, sqlArgs...)
		return e.returnSelect(
			oneValue.Elem(),
			err,
		)
	default:
		panic("not support such type " + resultType.String())
	}

	return nil
}
