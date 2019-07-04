package connect

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
)

var conn []*sql.DB

type MySql struct {
	User []string
	Password []string
	Host []string
	Port []string
	DBName []string
	Charset []string
	InterpolateParams []bool
	MaxOpenCoons []int
	connNumber int
}


func (db *MySql)SetConNumber(n int)*MySql{
	db.connNumber = n
	return db
}

func (db *MySql)GetConNumber()int{
	connNumber := 1
	if db.connNumber > 0 {
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
	connNumber := db.GetConNumber()
	var pass string
	if db.Password[connNumber]!="" {
		pass = fmt.Sprintf(":%s", db.Password[connNumber])
	} else {
		pass = ""
	}
	return pass
}

func (db *MySql)getCharset() string {
	connNumber := db.GetConNumber()
	var charset string
	if db.Charset[connNumber]!="" {
		charset = db.Charset[connNumber]
	} else {
		charset = "utf8"
	}

	return charset
}

func (db *MySql)getInterpolateParams() string{
	connNumber := db.GetConNumber()
	var param string
	if db.InterpolateParams[connNumber] {
		param = "true"
	} else {
		param = "false"
	}
	return param
}
func (db *MySql) GetMaxOpenCsr() int {
	connNumber := db.GetConNumber()
	var lifetime int
	if db.MaxOpenCoons[connNumber] > 0 {
		lifetime = db.MaxOpenCoons[connNumber]
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

