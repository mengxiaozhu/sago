package sago

import (
	"bytes"
	"database/sql"
	"github.com/jmoiron/sqlx"
	"log"
	"reflect"
	"strings"
	"text/template"
)

var ShowSQL = false

type SQLExecutor struct {
	Cache         Cache
	daoName       string
	Table         string
	FieldsString  string
	Fn            Fn
	Tpl           *template.Template
	db            *sql.DB
	DB            *sqlx.DB
	funcFactories []TemplateFuncFactory
	ReturnTypes   []reflect.Type
}

func NewSQLExecutor(table string, structTypeName string, returnTypes []reflect.Type, fn *Fn, tpl *template.Template, db *sql.DB, funcFactories []TemplateFuncFactory) *SQLExecutor {
	executor := &SQLExecutor{
		Table:         table,
		Fn:            *fn,
		ReturnTypes:   returnTypes,
		Tpl:           tpl,
		db:            db,
		daoName:       structTypeName,
		funcFactories: funcFactories,
	}
	driverName := "mysql"
	executor.DB = sqlx.NewDb(executor.db, driverName)
	if typ := findStructType(returnTypes[0]); typ != nil {
		names := executor.DB.Mapper.TypeMap(typ).Names
		fields := []string{}
		for v := range names {
			fields = append(fields, v)
		}
		executor.FieldsString = "`" + strings.Join(fields, "`,`") + "`"
	}
	return executor
}
func findStructType(typ reflect.Type) reflect.Type {
F:
	switch typ.Kind() {
	case reflect.Struct:
		return typ

	case reflect.Ptr, reflect.Slice:
		typ = typ.Elem()
		goto F
	default:
		return nil
	}
}
func (e *SQLExecutor) executeTpl(args []reflect.Value) (sql string, sqlArgs []interface{}, err error) {
	ctx := map[string]interface{}{}
	for i, v := range args {
		ctx[e.Fn.Args[i]] = v.Interface()
	}
	tpl, _ := e.Tpl.Clone()
	ctx["table"] = e.Table
	ctx["fields"] = e.FieldsString
	buf := bytes.NewBuffer(nil)

	fnCtx := &FnCtx{Args: []interface{}{}}

	fnMap := template.FuncMap{}

	for _, factory := range e.funcFactories {
		fnMap[factory.Name] = factory.Create(fnCtx)
	}

	tpl.Funcs(fnMap)

	err = tpl.Execute(buf, ctx)

	if err != nil {
		return "", nil, err
	}
	sql = buf.String()

	sqlArgs = fnCtx.Args
	if ShowSQL {
		log.Println(sql, sqlArgs)
	}
	return
}
