package main

import (
	"fmt"
	c "github.com/akula410/connect"
)

var connMySql c.MySql
func init(){
	connMySql.DBName = "test_db"
	connMySql.Host = "localhost"
	connMySql.User = "root"
	connMySql.Password = ""
	connMySql.Port = "3306"
	connMySql.Charset = "utf8"
	connMySql.InterpolateParams = true
	connMySql.MaxOpenCoons = 10
}

func main(){
	conn := connMySql.Connect()
	fmt.Println(conn)
}
