package connect

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
)

const DefaultConnName = "master"

var conn map[string]*sql.DB

type MySql struct {
	User string
	Password string
	Host string
	Port string
	DBName string
	Charset string
	InterpolateParams bool
	MaxOpenCoons int
	connNumber string
}


func (db *MySql)SetConNumber(n string)*MySql{
	db.connNumber = n
	return db
}

func (db *MySql)GetConNumber()string{
	connNumber := DefaultConnName
	if len(db.connNumber) > 0 {
		connNumber = db.connNumber
	}
	return connNumber
}

func (db *MySql) Connect() *sql.DB {
	connNumber := db.GetConNumber()

	if conn[connNumber]==nil {
		var err error
		conn[connNumber], err = sql.Open("mysql", db.csr())

		if err != nil {
			panic(err.Error())
		}


		conn[connNumber].SetMaxOpenConns(db.GetMaxOpenCsr())
		err = conn[connNumber].Ping()

		if err != nil {
			panic(err.Error())
		}
	}

	return conn[connNumber]
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
	connNumber := db.GetConNumber()
	if conn[connNumber]!=nil {
		err := conn[connNumber].Close()
		if err != nil {
			panic(err.Error())
		}
		conn[connNumber]=nil
	}
}

