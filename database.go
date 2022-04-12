/*
 * @Author: 周家建
 * @Mail: zhou_0611@163.com
 * @Date: 2021-07-27 19:02:39
 * @Description:
 */

package main

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
	"strings"
)

func instanceDB(str string) (*sql.DB, error) {
	sp := strings.Split(str, "://")
	if len(sp) == 2 {
		return sql.Open(sp[0], sp[1])
	} else {
		return sql.Open("mysql", str)
	}
}
