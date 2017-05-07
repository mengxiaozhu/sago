package sago

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/mengxiaozhu/linkerror"
	"os"
	"reflect"
	"strings"
	"text/template"
)

var DirError = errors.New("read dir wrong")
var ParseXMLError = errors.New("parse xml error")
var NoDBField = errors.New("no DB field in struct.")
var NoCacheInManager = errors.New("manager's Cache field is nil")
var BadCacheField = errors.New("Cache field of this struct is wrong type for sago. the field type must be pointer of the struct type")
var BadSQLTemplate = errors.New("bad sql")
var WrongTypeToMap = errors.New("map object must be pointer of struct")
var XMLMappedWrong = errors.New("XML mapped wrong")

type Cache interface {
	Set(dir string, key string, v interface{})
	Get(dir string, key string) (v interface{}, ok bool)
}

func New() *Manager {
	m := &Manager{
		files:         []*XMLRoot{},
		funcFactories: []TemplateFuncFactory{},
	}
	m.AddFunc(MethodName_Arg, argFunc)
	return m
}

type TemplateFunc func(interface{}) (string, error)

type TemplateFuncFactory struct {
	Name   string
	Create func(ctx *FnCtx) TemplateFunc
}
type Manager struct {
	Cache         Cache
	files         []*XMLRoot
	funcFactories []TemplateFuncFactory
	converted     bool
	fullNameMap   map[string]*SQLs
}

func (m *Manager) ScanDir(dirPath string) (e error) {
	m.converted = false
	dir, err := os.Open(dirPath)
	if err != nil {
		return linkerror.New(DirError, err.Error())
	}
	defer dir.Close()
	stat, err := dir.Stat()
	if err != nil {
		return linkerror.New(DirError, err.Error())
	}
	if !stat.IsDir() {
		return linkerror.New(DirError, dirPath+" not dir")
	}
	files, err := dir.Readdir(-1)
	if err != nil {
		return linkerror.New(DirError, err.Error())
	}
	for _, fileinfo := range files {
		if !fileinfo.IsDir() {
			name := fileinfo.Name()
			if strings.HasSuffix(name, ".sql.xml") {
				root, err := parseXML(dirPath + string(os.PathSeparator) + name)
				if err != nil {
					return linkerror.New(ParseXMLError, err.Error())
				}
				m.files = append(m.files, root)
			}
		}
	}
	return nil
}

func (m *Manager) Map(objs ...interface{}) error {
	if !m.converted {
		err := m.convert()
		if err != nil {
			return err
		}
	}

	for _, obj := range objs {
		err := m.mapFuncs(obj)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) convert() (err *linkerror.Error) {
	files := map[string]*XMLRoot{}
	for _, file := range m.files {
		var name string
		if file.Package == "" {
			name = file.Type
		} else {
			name = file.Package + "." + file.Type
		}
		if existFile, ok := files[name]; ok {
			files[name], err = combineXMLRoots(file, existFile)
			if err != nil {
				return
			}
		} else {
			files[name] = file
		}
	}
	fullNameSqls := map[string]*SQLs{}
	for name, v := range files {
		sqls := &SQLs{
			Package: v.Package,
			Type:    v.Type,
			Table:   v.Table,
		}
		sqls.Fns = map[string]*Fn{}
		insertByType("select", sqls.Fns, v.Selects)
		insertByType("execute", sqls.Fns, v.Executes)
		insertByType("insert", sqls.Fns, v.Inserts)
		fullNameSqls[name] = sqls
	}
	m.fullNameMap = fullNameSqls
	m.converted = true
	return nil
}
func insertByType(typ string, m map[string]*Fn, sqls []SQLContent) {
	for _, v := range sqls {
		m[v.Name] = &Fn{
			Name: v.Name,
			SQL:  strings.TrimSpace(v.SQL),
			Type: typ,
			Args: StrToArgs(v.Args),
		}
	}
}
func StrToArgs(str string) []string {
	splits := strings.Split(str, ",")
	result := []string{}
	for _, v := range splits {
		v = strings.TrimSpace(v)
		if v != "" {
			result = append(result, v)
		}
	}
	return result
}
func (m *Manager) getSQLs(typ reflect.Type) (sqls *SQLs, usedName string) {
	pkg := typ.PkgPath()
	typeName := typ.Name()
	sqls = m.fullNameMap[pkg+"."+typeName]
	if sqls == nil {
		sqls = m.fullNameMap[typeName]
		if sqls == nil {
			return nil, ""
		}
		usedName = typeName
	} else {
		usedName = pkg + "." + typeName
	}
	return sqls, usedName
}

func (m *Manager) AddFunc(name string, fnFactory func(ctx *FnCtx) (fn TemplateFunc)) {
	m.funcFactories = append(m.funcFactories, TemplateFuncFactory{Create: fnFactory, Name: name})
}
func (m *Manager) emptyFuncMap() template.FuncMap {
	fm := template.FuncMap{}
	for _, factory := range m.funcFactories {
		fm[factory.Name] = empty
	}
	return fm
}

func getDBFieldFromStruct(st reflect.Value) (DB *sql.DB, err *linkerror.Error) {
	dbValue := st.FieldByName("DB")
	if dbValue == emptyReflectValue {
		return nil, linkerror.New(NoDBField, st.String()+" must have a field named DB with *sql.DB type")
	}
	if dbValue.Type() != reflect.TypeOf(&sql.DB{}) {
		return nil, linkerror.New(NoDBField, st.String()+" must have a field named DB with *sql.DB type")
	}

	return dbValue.Interface().(*sql.DB), nil
}
func (m *Manager) mapCachedFuncs(structValue reflect.Value) (err *linkerror.Error) {

	cacheField := structValue.FieldByName("Cache")
	if cacheField == emptyReflectValue {
		return
	}
	if cacheField.Type().Kind() != reflect.Ptr {
		return linkerror.New(BadCacheField, "kind of Cache is "+cacheField.Type().Kind().String()+" ,but expected pointer")
	}
	if cacheField.Type().Elem() != structValue.Type() {
		return linkerror.New(BadCacheField, cacheField.Type().Elem().String()+" != "+structValue.Type().String())
	}
	if m.Cache == nil {
		return linkerror.New(NoCacheInManager, structValue.Type().String()+" cannot use cache because manager's Cache field is nil.")

	}
	cachedObject := reflect.New(structValue.Type())
	cacheField.Set(cachedObject)
	cachedObject.Elem().FieldByName("DB").Set(structValue.FieldByName("DB"))
	err = m.mapFns(true, cachedObject.Elem().Type(), cachedObject.Elem())
	if err != nil {
		return
	}
	return
}
func (m *Manager) mapFns(needCache bool, typ reflect.Type, value reflect.Value) (err *linkerror.Error) {
	// get sqls by type
	sqls, usedName := m.getSQLs(typ)
	if sqls == nil {
		return linkerror.New(XMLMappedWrong, "cannot found sqls to this type "+typ.PkgPath()+"."+typ.Name())
	}

	// get value's db field
	db, err := getDBFieldFromStruct(value)
	if err != nil {
		return err
	}
	// fill all func
	num := typ.NumField()

	for i := 0; i < num; i++ {
		f := typ.Field(i)
		if f.Type.Kind() == reflect.Func {
			fn, err := m.generateFunc(needCache, usedName, sqls.Fns[f.Name], f, db, sqls.Table)
			if err != nil {
				return err
			}
			value.Field(i).Set(fn)
		}
	}

	return
}
func (m *Manager) mapFuncs(obj interface{}) (err *linkerror.Error) {
	value := reflect.ValueOf(obj)

	typ := reflect.TypeOf(obj)
	// type check
	if typ.Kind() != reflect.Ptr {
		return linkerror.New(WrongTypeToMap, "but got "+typ.Kind().String()+" -> "+typ.String())
	}
	// change pointer type to struct type
	typ = typ.Elem()
	value = value.Elem()
	err = m.mapFns(false, typ, value)
	if err != nil {
		return err
	}
	err = m.mapCachedFuncs(value)
	if err != nil {
		return err
	}
	return
}

// generate func
func (m *Manager) generateFunc(needCache bool, usedName string, fn *Fn, f reflect.StructField, db *sql.DB, table string) (generatedFunc reflect.Value, err *linkerror.Error) {
	if fn == nil {
		return emptyReflectValue, linkerror.New(XMLMappedWrong, "cannot found func "+f.Name+" mapped sql")
	}
	if len(fn.Args) != f.Type.NumIn() {
		return emptyReflectValue, linkerror.New(XMLMappedWrong, fmt.Sprint(f.Name, " Args number is wrong , expected ", f.Type.NumIn(), " but xml defined ", fn.Args))
	}
	tpl, tplErr := template.New(fn.Name).Funcs(m.emptyFuncMap()).Parse(fn.SQL)
	if tplErr != nil {
		return emptyReflectValue, linkerror.New(BadSQLTemplate, tplErr.Error()+":"+fn.SQL)
	}
	out := f.Type.NumOut()
	returnTypes := make([]reflect.Type, 0, out)
	for n := 0; n < out; n++ {
		returnTypes = append(returnTypes, f.Type.Out(n))
	}
	sqlExecutor := NewSQLExecutor(table, usedName, returnTypes, fn, tpl, db, m.funcFactories)
	switch fn.Type {
	case "select":
		if needCache {
			sqlExecutor.Cache = m.Cache
			generatedFunc = reflect.MakeFunc(f.Type, sqlExecutor.SelectCache)
		} else {
			generatedFunc = reflect.MakeFunc(f.Type, sqlExecutor.Select)
		}
	case "insert":
		generatedFunc = reflect.MakeFunc(f.Type, sqlExecutor.Insert)
	case "execute":
		generatedFunc = reflect.MakeFunc(f.Type, sqlExecutor.Execute)
	}
	return
}
