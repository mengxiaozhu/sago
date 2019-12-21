package sago

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
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
		files:         []*File{},
		funcFactories: []TemplateFuncFactory{},
	}
	m.AddFunc(MethodNameArg, argFunc)
	m.AddFunc(MethodInArg, inFunc)
	return m
}

type TemplateFunc func(interface{}) (string, error)

type TemplateFuncFactory struct {
	Name   string
	Create func(ctx *FnCtx) TemplateFunc
}
type Central struct {
	Cache         Cache
	files         []*File
	funcFactories []TemplateFuncFactory
	converted     bool
	fullNameMap   map[string]*SQLSet
}

const xmlSuffix = ".sql.xml"
const yamlSuffix = ".sql.yaml"

// 扫描文件下以所有以 .sql.xml 结尾的文件
// <sago>
// 	<package></package>
//	<type></type>
//	<table></table>
//	<select name="FindByName" args="name">
// 		select {{.fields}} from {{.table}} where `name` = {{arg .name}}
// 	</select>
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
			switch {
			case strings.HasSuffix(name, xmlSuffix):
				root, err := parseXML(filepath.Join(dirPath, name))
				if err != nil {
					return linkerror.New(Xml, err.Error())
				}
				m.files = append(m.files, root)
			case strings.HasSuffix(name, yamlSuffix):
				root, err := parseYAML(filepath.Join(dirPath, name))
				if err != nil {
					return linkerror.New(YAML, err.Error())
				}
				m.files = append(m.files, root)
			}
		}
	}
	return nil
}

// 参数的基本类型必须是 Ptr
// 读取变量并注入配置文件中配置的方法
func (m *Central) Map(daoObjects ...interface{}) error {
	if !m.converted {
		err := m.convert()
		if err != nil {
			return err
		}
	}
	for _, dao := range daoObjects {
		err := m.mapMethods(dao)
		if err != nil {
			return err
		}
	}
	return nil
}

// 根据xml生成配置
func (m *Central) convert() (err *linkerror.Error) {
	files := map[string]*File{}
	// 去重合并
	for _, f := range m.files {
		name := f.Name()
		if existFile, ok := files[name]; ok {
			files[name], err = combineFiles(f, existFile)
			if err != nil {
				return
			}
		} else {
			files[name] = f
		}
	}
	fullNameSQLs := map[string]*SQLSet{}
	for name, xml := range files {
		sqls := &SQLSet{
			Package: xml.Package,
			Type:    xml.Type,
			Table:   xml.Table,
		}
		sqls.Functions = map[string]*Fn{}
		insertByType("select", sqls.Functions, xml.Selects)
		insertByType("execute", sqls.Functions, xml.Executes)
		insertByType("insert", sqls.Functions, xml.Inserts)
		fullNameSQLs[name] = sqls
	}
	m.fullNameMap = fullNameSQLs
	m.converted = true
	return nil
}

func insertByType(typ string, m map[string]*Fn, sqls []SQLContent) {
	for _, v := range sqls {
		m[v.Name] = &Fn{
			Name: v.Name,
			SQL:  strings.TrimSpace(v.SQL),
			Type: typ,
			Args: strToArgs(v.Args),
		}
	}
}

func strToArgs(str string) []string {
	splits := strings.Split(str, ",")
	var result []string
	for _, v := range splits {
		v = strings.TrimSpace(v)
		if v != "" {
			result = append(result, v)
		}
	}
	return result
}

func (m *Central) getSQLSet(typ reflect.Type) (sqlSet *SQLSet, name string) {
	pkg := typ.PkgPath()
	typeName := typ.Name()
	name = pkg + "." + typeName
	if sqlSet = m.fullNameMap[name]; sqlSet != nil {
		return sqlSet, name
	}
	if sqlSet = m.fullNameMap[typeName]; sqlSet != nil {
		name = typeName
		return sqlSet, name
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

func (m *Central) mapCachedMethods(structValue reflect.Value) (err *linkerror.Error) {
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
	err = m.injectFuncs(true, cachedObject.Elem().Type(), cachedObject.Elem())
	if err != nil {
		return
	}
	return
}

func (m *Central) injectFuncs(needCache bool, typ reflect.Type, value reflect.Value) (err *linkerror.Error) {
	sqlSet, name := m.getSQLSet(typ)
	if sqlSet == nil {
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
			fn, err := m.generateFunc(needCache, name, sqlSet.Functions[f.Name], f, db, sqlSet.Table)
			if err != nil {
				return err
			}
			value.Field(i).Set(fn)
		}
	}
	return
}

func (m *Central) mapMethods(obj interface{}) (err *linkerror.Error) {
	value := reflect.ValueOf(obj)
	typ := value.Type()
	// 检查基本类型是否为Ptr
	if typ.Kind() != reflect.Ptr {
		return linkerror.New(WrongTypeToMap, "but got "+typ.Kind().String()+" -> "+typ.String())
	}
	// 取得具体对象
	value = value.Elem()
	typ = typ.Elem()
	err = m.injectFuncs(false, typ, value)
	if err != nil {
		return err
	}
	err = m.mapCachedMethods(value)
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
