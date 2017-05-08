package sago

import "reflect"

func (s *SQLExecutor) Execute(args []reflect.Value) (results []reflect.Value) {
	sqlstring, sqlargs, err := s.executeTpl(args)
	if err != nil {
		return s.returnError(err)
	}
	rs, err := s.DB.Exec(sqlstring, sqlargs...)
	if err != nil {
		return s.returnError(err)
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
