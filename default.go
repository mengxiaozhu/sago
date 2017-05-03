package sago

var DefaultManager = New()

func ScanDir(dirPath string) (e error) {
	return DefaultManager.ScanDir(dirPath)
}

func Map(objs ...interface{}) error {
	return DefaultManager.Map(objs...)
}
func MustMap(obj interface{}) {
	err := Map(obj)
	if err != nil {
		panic(err)
	}
}

func AddFunc(name string, fnFactory func(ctx *FnCtx) (fn TemplateFunc)) {
	DefaultManager.AddFunc(name, fnFactory)
}
