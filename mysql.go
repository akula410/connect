package connect

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
)

const DefaultConnName = "master"

var conn = make(map[string]*sql.DB)

type MySql struct {
	User string
	Password string
	Host string
	Port string
	DBName string
	Charset string
	InterpolateParams bool
	MaxOpenCoons int
	connName string
}


func (db *MySql)SetConnName(n string)*MySql{
	db.connName = n
	return db
}

func (db *MySql)GetConnName()string{
	connNumber := DefaultConnName
	if len(db.connName) > 0 {
		connNumber = db.connName
	}
	return connNumber
}

func (db *MySql) Connect() *sql.DB {
	connName := db.GetConnName()

	if conn[connName]==nil {
		var err error
		conn[connName], err = sql.Open("mysql", db.csr())

		if err != nil {
			panic(err.Error())
		}


		conn[connName].SetMaxOpenConns(db.GetMaxOpenCsr())
		err = conn[connName].Ping()

		if err != nil {
			panic(err.Error())
		}
	}

	return conn[connName]
}


func (db *MySql) csr() string{
	dsn := fmt.Sprintf("%s%s@tcp(%s:%s)/%s?", db.User, db.getPass(), db.Host, db.Port, db.DBName)
	dsn += fmt.Sprintf("&charset=%s", db.getCharset())
	dsn += fmt.Sprintf("&interpolateParams=%s", db.getInterpolateParams())
	return dsn
}


func (db *MySql)getPass() string{
	var pass string
	if db.Password!="" {
		pass = fmt.Sprintf(":%s", db.Password)
	} else {
		pass = ""
	}
	return pass
}

func (db *MySql)getCharset() string {
	var charset string
	if db.Charset!="" {
		charset = db.Charset
	} else {
		charset = "utf8"
	}

	return charset
}

func (db *MySql)getInterpolateParams() string{
	var param string
	if db.InterpolateParams {
		param = "true"
	} else {
		param = "false"
	}
	return param
}
func (db *MySql) GetMaxOpenCsr() int {
	var lifetime int
	if db.MaxOpenCoons > 0 {
		lifetime = db.MaxOpenCoons
	} else {
		lifetime = 10
	}

	return lifetime
}

func (db *MySql) Close(){
	connName := db.GetConnName()
	if conn[connName]!=nil {
		err := conn[connName].Close()
		if err != nil {
			panic(err.Error())
		}
		conn[connName]=nil
	}
}

