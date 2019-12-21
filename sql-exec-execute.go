package sago

import (
	"reflect"
)

var (
	emptyError     error
	emptyErrorType = reflect.TypeOf(&emptyError).Elem()
)

func (e *SQLExecutor) returnError(err error) (results []reflect.Value) {
	results = []reflect.Value{}
	for _, typ := range e.ReturnTypes {
		if typ == emptyErrorType {
			results = append(results, reflect.ValueOf(&err).Elem())
		} else {
			results = append(results, reflect.Zero(typ))
		}
	}
	return results
}

func (e *SQLExecutor) Insert(args []reflect.Value) (results []reflect.Value) {
	sqlText, sqlArgs, err := e.executeTpl(args)

	if err != nil {
		return e.returnError(err)
	}
	rs, err := e.DB.Exec(sqlText, sqlArgs...)

	if err != nil {
		return e.returnError(err)
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
	var nilError error
	affected, _ := rs.RowsAffected()
	if e.ReturnTypes[0].Kind() == reflect.Int64 {
		return []reflect.Value{
			reflect.ValueOf(affected),
			reflect.ValueOf(&nilError).Elem(),
		}
	}
	if e.ReturnTypes[0].Kind() == reflect.Int {
		return []reflect.Value{
			reflect.ValueOf(int(affected)),
			reflect.ValueOf(&nilError).Elem(),
		}
	}
	if len(e.ReturnTypes) == 1 {
		return e.returnError(nilError)
	}
	panic("NOT SUPPORT")
}
