package connection

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type Connection interface {
	Exec(query string, args ...any) (sql.Result, error)
	Select(dest interface{}, query string, args ...interface{}) error
	Get(dest interface{}, query string, args ...interface{}) error
}

type ConnectionFactory interface {
	New() (Connection, error)
}

type MySQLConnectionFactory struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
}

func NewMySQLConnectionFactory(host string, port int, username string, password string, database string) *MySQLConnectionFactory {
	return &MySQLConnectionFactory{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		Database: database,
	}
}

func (f *MySQLConnectionFactory) New() (Connection, error) {
	return sqlx.Connect("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", f.Username, f.Password, f.Host, f.Port, f.Database))
}
