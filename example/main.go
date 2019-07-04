package main

import (
	"fmt"
	c "github.com/akula410/connect"
)

var connMySql c.MySql

var connMySql2 c.MySql

func init(){
	connMySql.DBName = "css"
	connMySql.Host = "localhost"
	connMySql.User = "root"
	connMySql.Password = ""
	connMySql.Port = "3306"
	connMySql.Charset = "utf8"
	connMySql.InterpolateParams = true
	connMySql.MaxOpenCoons = 10

	connMySql2.DBName = ""
	connMySql2.Host = "localhost"
	connMySql2.User = "root"
	connMySql2.Password = ""
	connMySql2.Port = "3306"
	connMySql2.Charset = "utf8"
	connMySql2.InterpolateParams = true
	connMySql2.MaxOpenCoons = 10
}

func main(){
	conn := connMySql.Connect()
	fmt.Println(conn)
	fmt.Println(connMySql.DBName)
	fmt.Println(connMySql.GetConnName())

	conn2 := connMySql2.SetConnName("slave").Connect()
	fmt.Println(conn2)
	fmt.Println(connMySql2.DBName)
	fmt.Println(connMySql2.GetConnName())


}
