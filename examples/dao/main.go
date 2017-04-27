package main

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/mengxiaozhu/sago"
	"log"
)

type User struct {
	Id   int    `db:"id"`
	Name string `db:"name"`
}

type UserDao struct {
	DB         *sql.DB
	Cache      *UserDao
	FindByName func(name string) ([]User, error)
	FindAll    func() ([]User, error)
}

// just for demo,don't use in your project
type UnSafeMapCache struct {
	cache map[string]interface{}
}

func (c *UnSafeMapCache) Set(dir string, key string, v interface{}) {
	c.cache[dir+key] = v
}

func (c *UnSafeMapCache) Get(dir string, key string) (v interface{}, ok bool) {
	v, ok = c.cache[dir+key]
	return v, ok
}

func main() {
	db, err := sql.Open("mysql", "root:123456@tcp(127.0.0.1:3306)/sago?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	dao := &UserDao{
		DB: db,
	}
	sago.ShowSQL = true

	manager := sago.New()
	// scan dir's all *.sql.xml
	err = manager.ScanDir("./")

	if err != nil {
		log.Fatal(err)
	}

	manager.Cache = &UnSafeMapCache{
		cache: map[string]interface{}{},
	}
	err = manager.Map(dao)
	if err != nil {
		log.Fatal(err)
	}

	// find by name
	list, err := dao.FindByName("bar")
	if err != nil {
		log.Fatal(err)
	}
	log.Println(list)
	// find by name with cache
	list, err = dao.Cache.FindByName("foo")
	if err != nil {
		log.Fatal(err)
	}
	log.Println(list)

	// find by name with cache
	list, err = dao.Cache.FindByName("foo")
	if err != nil {
		log.Fatal(err)
	}
	log.Println(list)

	f, _ := dao.FindAll()
	log.Println(f)

}