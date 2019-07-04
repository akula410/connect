package main

import (
	"fmt"
	c "github.com/akula410/connect"
)

var connMySql c.MySql

var connMySql2 c.MySql

func init(){
	connMySql.DBName[1] = "test_db"
	connMySql.Host[1] = "localhost"
	connMySql.User[1] = "root"
	connMySql.Password[1] = ""
	connMySql.Port[1] = "3306"
	connMySql.Charset[1] = "utf8"
	connMySql.InterpolateParams[1] = true
	connMySql.MaxOpenCoons[1] = 10

	connMySql2.DBName[2] = ""
	connMySql2.Host[2] = "localhost"
	connMySql2.User[2] = "root"
	connMySql2.Password[2] = ""
	connMySql2.Port[2] = "3306"
	connMySql2.Charset[2] = "utf8"
	connMySql2.InterpolateParams[2] = true
	connMySql2.MaxOpenCoons[2] = 10
}

func main(){
	conn := connMySql.Connect()
	fmt.Println(conn)

	conn2 := connMySql.SetConNumber("slave").Connect()
	fmt.Println(conn2)


}
