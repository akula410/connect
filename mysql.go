package mysql

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
)

var conn *sql.DB

type MySql struct {
	User string
	Password string
	Host string
	Port string
	DBName string
	Charset string
	InterpolateParams bool
	MaxOpenCoons int
}





func (db *MySql) Connect() *sql.DB {

	if conn==nil {
		var err error
		conn, err = sql.Open("mysql", db.csr())

		if err != nil {
			panic(err.Error())
		}
		conn.SetMaxOpenConns(db.GetMaxOpenCsr())
		err = conn.Ping()

		if err != nil {
			panic(err.Error())
		}
	}

	return conn
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
	if conn!=nil {
		err := conn.Close()
		if err != nil {
			panic(err.Error())
		}
		conn=nil
	}
}

