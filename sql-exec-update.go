package sago

import "reflect"

func (s *SQLExecutor) Execute(args []reflect.Value) (results []reflect.Value) {
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
	var emptyError error
	affected, _ := rs.RowsAffected()
	return []reflect.Value{
		reflect.ValueOf(affected),
		reflect.ValueOf(&emptyError).Elem(),
	}
}
