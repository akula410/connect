# connect
Mysql connection example

package Configs

import (
	"ModuleModel/Constructor/DB/Driver"
)

var	ConnMysql Driver.MySql

func init(){
	ConnMysql.DBName = ""
	ConnMysql.Host = ""
	ConnMysql.User = ""
	ConnMysql.Password = ""
	ConnMysql.Port = ""
	ConnMysql.Charset = "utf8"
	ConnMysql.InterpolateParams = true
	ConnMysql.MaxOpenCoons = 10
}
