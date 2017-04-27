package sago

import (
	"reflect"
)

func (s *SQLExecutor) Insert(args []reflect.Value) (results []reflect.Value) {
	sqlstring, sqlargs, err := s.executeTpl(args)

	if err != nil {
		return []reflect.Value{
			reflect.Zero(s.ReturnTypes[0]),
			reflect.ValueOf(&err).Elem(),
		}
	}
	rs, err := s.DB.Exec(sqlstring, sqlargs...)

	if err != nil {
		return []reflect.Value{
			reflect.Zero(s.ReturnTypes[0]),
			reflect.ValueOf(&err).Elem(),
		}
	}
	firstArg := args[0]
	if firstArg.Kind() == reflect.Ptr && firstArg.Elem().Kind() == reflect.Struct {
		idField := firstArg.Elem().FieldByName("Id")
		empty := reflect.Value{}
		if idField == empty {
			idField = firstArg.Elem().FieldByName("ID")
		}
		if idField != empty {
			id, _ := rs.LastInsertId()
			idField.SetInt(id)
		}
	}
	var emptyError error
	affected, _ := rs.RowsAffected()
	return []reflect.Value{
		reflect.ValueOf(affected),
		reflect.ValueOf(&emptyError).Elem(),
	}
}
