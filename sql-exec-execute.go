package sago

import (
	"reflect"
)

var (
	emptyError     error
	emptyErrorType = reflect.TypeOf(&emptyError).Elem()
)

func (s *SQLExecutor) returnError(err error) (results []reflect.Value) {
	results = []reflect.Value{}
	for _, typ := range s.ReturnTypes {
		if typ == emptyErrorType {
			results = append(results, reflect.ValueOf(&err).Elem())
		} else {
			results = append(results, reflect.Zero(s.ReturnTypes[0]))
		}
	}
	return results
}
func (s *SQLExecutor) Insert(args []reflect.Value) (results []reflect.Value) {
	sqlstring, sqlargs, err := s.executeTpl(args)

	if err != nil {
		return s.returnError(err)
	}
	rs, err := s.DB.Exec(sqlstring, sqlargs...)

	if err != nil {
		return s.returnError(err)
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
	if s.ReturnTypes[0].Kind() == reflect.Int64 {
		return []reflect.Value{
			reflect.ValueOf(affected),
			reflect.ValueOf(&emptyError).Elem(),
		}
	}
	if s.ReturnTypes[0].Kind() == reflect.Int {
		return []reflect.Value{
			reflect.ValueOf(int(affected)),
			reflect.ValueOf(&emptyError).Elem(),
		}
	}
	if len(s.ReturnTypes) == 1 {
		return s.returnError(emptyError)
	}
	panic("NOT SUPPORT")
}
