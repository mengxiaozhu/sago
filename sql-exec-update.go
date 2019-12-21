package sago

import "reflect"

func (e *SQLExecutor) Execute(args []reflect.Value) (results []reflect.Value) {
	sqlText, sqlArgs, err := e.executeTpl(args)
	if err != nil {
		return e.returnError(err)
	}
	rs, err := e.DB.Exec(sqlText, sqlArgs...)
	if err != nil {
		return e.returnError(err)
	}
	var emptyError error
	affected, _ := rs.RowsAffected()
	if e.ReturnTypes[0].Kind() == reflect.Int64 {
		return []reflect.Value{
			reflect.ValueOf(affected),
			reflect.ValueOf(&emptyError).Elem(),
		}
	}
	if e.ReturnTypes[0].Kind() == reflect.Int {
		return []reflect.Value{
			reflect.ValueOf(int(affected)),
			reflect.ValueOf(&emptyError).Elem(),
		}
	}
	if len(e.ReturnTypes) == 1 {
		return e.returnError(emptyError)
	}
	panic("NOT SUPPORT")
}
