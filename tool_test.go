package sago

import (
	"testing"
	"text/template"
	"bytes"
)

func TestInStrs(t *testing.T) {
	ctx := &FnCtx{Args: []interface{}{}}

	tpl := template.Must(template.New("").Funcs(template.FuncMap{
		"in": inFunc(ctx),
	}).Parse(`select * from table where id {{in .list}}`))
	buf := bytes.NewBuffer(nil)

	tpl.Execute(buf, map[string]interface{}{
		"list": []string{
			"HEADER", "BODY",
		},
	})
	if len(ctx.Args)!=2{
		t.Fail()
	}
	if buf.String()!="select * from table where id in (?,?)"{
		t.Fail()
	}
	t.Log(buf.String(),ctx.Args)
}

func TestInInts(t *testing.T) {
	ctx := &FnCtx{Args: []interface{}{}}

	tpl := template.Must(template.New("").Funcs(template.FuncMap{
		"in": inFunc(ctx),
	}).Parse(`select * from table where id {{in .list}}`))
	buf := bytes.NewBuffer(nil)

	tpl.Execute(buf, map[string]interface{}{
		"list": []int{
			1, 2,
		},
	})
	if len(ctx.Args)!=2{
		t.Fail()
	}
	if buf.String()!="select * from table where id in (?,?)"{
		t.Fail()
	}
	t.Log(buf.String(),ctx.Args)
}


func TestInFloats(t *testing.T) {
	ctx := &FnCtx{Args: []interface{}{}}

	tpl := template.Must(template.New("").Funcs(template.FuncMap{
		"in": inFunc(ctx),
	}).Parse(`select * from table where id {{in .list}}`))
	buf := bytes.NewBuffer(nil)

	tpl.Execute(buf, map[string]interface{}{
		"list": []float64{
			0.2, 0.1,
		},
	})
	if len(ctx.Args)!=2{
		t.Fail()
	}
	if buf.String()!="select * from table where id in (?,?)"{
		t.Fail()
	}
	t.Log(buf.String(),ctx.Args)
}


func TestInInterfaces(t *testing.T) {
	ctx := &FnCtx{Args: []interface{}{}}

	tpl := template.Must(template.New("").Funcs(template.FuncMap{
		"in": inFunc(ctx),
	}).Parse(`select * from table where id {{in .list}}`))
	buf := bytes.NewBuffer(nil)

	tpl.Execute(buf, map[string]interface{}{
		"list": []interface{}{
			"HEADER", "BODY",
		},
	})
	if len(ctx.Args)!=2{
		t.Fail()
	}
	if buf.String()!="select * from table where id in (?,?)"{
		t.Fail()
	}
	t.Log(buf.String(),ctx.Args)
}
