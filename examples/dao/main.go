package main

import (
	"database/sql"
	"log"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/mengxiaozhu/sago"
)

var logger = log.New(os.Stdout, "[examples] ", log.Lshortfile)

// 仅供测试
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

type User struct {
	Id   int    `db:"id"`
	Name string `db:"name"`
}

type UserDao struct {
	DB         *sql.DB                           // 名称必须是DB
	Cache      *UserDao                          // 名字必须是Cache
	FindByName func(name string) ([]User, error) // select {{.fields}} from {{.table}} where `name` = {{arg .name}}
	FindByID   func(id int) ([]User, error)      // select {{.fields}} from {{.table}} where `id` = {{arg .name}}
	FindAll    func() ([]User, error)            // select {{.fields}} from {{.table}}
}

func main() {
	var (
		user     = os.Getenv("mysql-user")     // 本地数据库默认 root
		password = os.Getenv("mysql-password") // 本地数据库默认 无
	)
	db, err := sql.Open("mysql", user+":"+password+"@tcp(127.0.0.1:3306)/?parseTime=true")
	if err != nil {
		logger.Fatal(err)
		return
	}
	{
		// example mysql 如果不存在就初始化
		// 新建库 sago
		// 新建表 user
		// 插入两条数据
		// id, name
		// 1 , "foo"
		// 2 , "bar"
		if _, err = db.Exec(`CREATE SCHEMA IF NOT EXISTS sago;`); err != nil {
			logger.Fatalln(err)
			return
		}
		_, err = db.Exec(`CREATE TABLE sago.user (id INT AUTO_INCREMENT PRIMARY KEY,name VARCHAR(11) NOT NULL);`)
		if err == nil {
			_, err := db.Exec(`INSERT INTO sago.user (name) VALUES ('foo'),('bar');`)
			if err != nil {
				logger.Fatalln(err)
				return
			}
		} else if !strings.HasPrefix(err.Error(), "Error 1050") {
			logger.Fatalln(err)
			return
		}
	}
	dao := &UserDao{
		DB: db,
	}

	sago.ShowSQL = true

	central := sago.New()
	// 读取配置文件
	err = central.ScanDir("./examples/dao")
	if err != nil {
		logger.Fatal(err)
		return
	}

	// 一个简陋的实现了Cache接口的线程不安全内存缓存,用于缓存select结果
	central.Cache = &UnSafeMapCache{
		cache: map[string]interface{}{},
	}
	err = central.Map(dao)
	if err != nil {
		logger.Fatal(err)
		return
	}

	list, err := dao.FindByName("bar")
	if err != nil {
		logger.Fatal(err)
		return
	}
	logger.Println(list)

	list, err = dao.FindByID(1)
	if err != nil {
		logger.Fatal(err)
		return
	}
	logger.Println(list)

	list, err = dao.Cache.FindByName("bar")
	if err != nil {
		logger.Fatal(err)
		return
	}
	logger.Println(list)

	f, err := dao.FindAll()
	if err != nil {
		logger.Fatal(err)
		return
	}
	logger.Println(f)
}
