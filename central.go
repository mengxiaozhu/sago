package sago

import (
	"database/sql"
	"fmt"
	"os"
	"reflect"
	"strings"
	"text/template"

	"github.com/mengxiaozhu/linkerror"
)

type Cache interface {
	Set(dir string, key string, v interface{})
	Get(dir string, key string) (v interface{}, ok bool)
}

func New() *Central {
	m := &Central{
		xmls:          []*XMLRoot{},
		funcFactories: []TemplateFuncFactory{},
	}
	m.AddFunc(MethodName_Arg, argFunc)
	m.AddFunc(MethodIn_Arg, inFunc)
	return m
}

type TemplateFunc func(interface{}) (string, error)

type TemplateFuncFactory struct {
	Name   string
	Create func(ctx *FnCtx) TemplateFunc
}
type Central struct {
	Cache         Cache
	xmls          []*XMLRoot
	funcFactories []TemplateFuncFactory
	converted     bool
	fullNameMap   map[string]*SQLs
}

// 扫描文件下以所有以 .sql.xml 结尾的文件
// <sago>
// 	<package></package>
//	<type></type>
//	<table></table>
//	<select name="FindByName" args="name">
// 		select {{.fields}} from {{.table}} where `name` = {{arg .name}}
// 	</select> 可重复
//	<execute></execute>
//	<insert></insert>
// </sago>
func (m *Central) ScanDir(dirPath string) (e error) {
	dir, err := os.Open(dirPath)
	if err != nil {
		return linkerror.New(Dir, err.Error())
	}
	defer dir.Close()
	stat, err := dir.Stat()
	if err != nil {
		return linkerror.New(Dir, err.Error())
	}
	if !stat.IsDir() {
		return linkerror.New(Dir, dirPath+" not dir")
	}
	files, err := dir.Readdir(-1)
	if err != nil {
		return linkerror.New(Dir, err.Error())
	}
	for _, fileInfo := range files {
		if !fileInfo.IsDir() {
			name := fileInfo.Name()
			if strings.HasSuffix(name, ".sql.xml") {
				root, err := parseXML(dirPath + string(os.PathSeparator) + name)
				if err != nil {
					return linkerror.New(Xml, err.Error())
				}
				m.xmls = append(m.xmls, root)
			}
		}
	}
	return nil
}

// 参数的基本类型必须是 Ptr
// 读取变量并注入配置文件中配置的方法
func (m *Central) Map(daos ...interface{}) error {
	if !m.converted {
		err := m.convert()
		if err != nil {
			return err
		}
	}
	for _, dao := range daos {
		err := m.mapFuncs(dao)
		if err != nil {
			return err
		}
	}
	return nil
}

// 根据xml生成配置
func (m *Central) convert() (err *linkerror.Error) {
	xmls := map[string]*XMLRoot{}
	// 去重合并
	for _, xml := range m.xmls {
		name := xml.Name()
		if existFile, ok := xmls[name]; ok {
			xmls[name], err = combineXMLRoots(xml, existFile)
			if err != nil {
				return
			}
		} else {
			xmls[name] = xml
		}
	}
	fullNameSqls := map[string]*SQLs{}
	for name, xml := range xmls {
		sqls := &SQLs{
			Package: xml.Package,
			Type:    xml.Type,
			Table:   xml.Table,
		}
		sqls.Fns = map[string]*Fn{}
		insertByType("select", sqls.Fns, xml.Selects)
		insertByType("execute", sqls.Fns, xml.Executes)
		insertByType("insert", sqls.Fns, xml.Inserts)
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
func (m *Central) getSQLs(typ reflect.Type) (sqls *SQLs, name string) {
	pkg := typ.PkgPath()
	typeName := typ.Name()
	name = pkg + "." + typeName
	if sqls = m.fullNameMap[name]; sqls != nil {
		return sqls, name
	}
	if sqls = m.fullNameMap[typeName]; sqls != nil {
		name = typeName
		return sqls, name
	}
	return nil, ""
}

func (m *Central) AddFunc(name string, fnFactory func(ctx *FnCtx) (fn TemplateFunc)) {
	m.funcFactories = append(m.funcFactories, TemplateFuncFactory{Create: fnFactory, Name: name})
}
func (m *Central) emptyFuncMap() template.FuncMap {
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
func (m *Central) mapCachedFuncs(structValue reflect.Value) (err *linkerror.Error) {

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
func (m *Central) mapFns(needCache bool, typ reflect.Type, value reflect.Value) (err *linkerror.Error) {
	sqls, name := m.getSQLs(typ)
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
			fn, err := m.generateFunc(needCache, name, sqls.Fns[f.Name], f, db, sqls.Table)
			if err != nil {
				return err
			}
			value.Field(i).Set(fn)
		}
	}

	return
}
func (m *Central) mapFuncs(obj interface{}) (err *linkerror.Error) {
	value := reflect.ValueOf(obj)
	typ := value.Type()
	// 检查基本类型是否为Ptr
	if typ.Kind() != reflect.Ptr {
		return linkerror.New(WrongTypeToMap, "but got "+typ.Kind().String()+" -> "+typ.String())
	}
	// 取得具体对象
	value = value.Elem()
	typ = typ.Elem()
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
func (m *Central) generateFunc(needCache bool, usedName string, fn *Fn, f reflect.StructField, db *sql.DB, table string) (generatedFunc reflect.Value, err *linkerror.Error) {
	if fn == nil {
		return emptyReflectValue, linkerror.New(XMLMappedWrong, "cannot found func "+f.Name+" mapped sql")
	}
	if len(fn.Args) != f.Type.NumIn() {
		return emptyReflectValue, linkerror.New(XMLMappedWrong, fmt.Sprint(f.Name, " Args number is wrong , expected ", f.Type.NumIn(), " but xml defined ", fn.Args, "length:", len(fn.Args)))
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
