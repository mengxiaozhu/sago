package sago

import (
	"reflect"
)

func Copy(value reflect.Value) reflect.Value {
	switch value.Type().Kind() {
	case reflect.Struct:
		cp := reflect.New(value.Type())
		cp.Elem().Set(value)
		return cp.Elem()
	case reflect.Slice:
		length := value.Len()
		slice := reflect.MakeSlice(value.Type(), 0, length)
		for i := 0; i < length; i++ {
			slice = reflect.Append(slice, Copy(value.Index(i)))
		}
		return slice
	case reflect.Ptr:
		cp := reflect.New(value.Elem().Type())
		cp.Elem().Set(Copy(value.Elem()))
		return cp
	}
	return emptyReflectValue
}
