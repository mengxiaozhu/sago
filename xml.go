package sago

import (
	"encoding/xml"
	"errors"
	"github.com/mengxiaozhu/linkerror"
)

type SQLContent struct {
	Name string `xml:"name,attr"`
	SQL  string `xml:",chardata"`
	Args string `xml:"args,attr"`
}
type XMLRoot struct {
	XMLName  xml.Name     `xml:"sago"`
	Package  string       `xml:"package"`
	Type     string       `xml:"type"`
	Table    string       `xml:"table"`
	Selects  []SQLContent `xml:"select"`
	Executes []SQLContent `xml:"execute"`
	Inserts  []SQLContent `xml:"insert"`
}

func (x XMLRoot) Name() string {
	if x.Package == "" {
		return x.Type
	}
	return x.Package + "." + x.Type
}

type Fn struct {
	Name string
	Type string
	SQL  string
	Args []string
}

type SQLs struct {
	Package string
	Type    string
	Table   string
	Fns     map[string]*Fn
}

var CombineConflict = errors.New("combine conflict")

func combineXMLRoots(r1, r2 *XMLRoot) (r *XMLRoot, err *linkerror.Error) {
	r = &XMLRoot{Table: r1.Table}
	if r2.Table != r1.Table {
		return nil, linkerror.New(CombineConflict, r.Package+"."+r.Type+" conflict "+r2.Table+"!="+r1.Table)
	}
	r.Executes = combineSQLContent(r1.Executes, r2.Executes)
	r.Selects = combineSQLContent(r1.Selects, r2.Selects)
	r.Inserts = combineSQLContent(r1.Inserts, r2.Inserts)
	return

}
func combineSQLContent(c1, c2 []SQLContent) []SQLContent {
	sqls := make([]SQLContent, 0, len(c1)+len(c2))
	sqls = append(sqls, c1...)
	sqls = append(sqls, c2...)
	return sqls
}
