package db

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

func ConnectClickHouse(url string, database string, user string, password string) (driver.Conn, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{url},
		Auth: clickhouse.Auth{
			Database: database,
			Username: user,
			Password: password,
		},
		Debug: false,
		// Debugf: func(format string, v ...interface{}) {
		// 	fmt.Printf(format, v...)
		// },
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
		DialTimeout:  time.Second * 5,
		MaxOpenConns: 10,
		MaxIdleConns: 5,
	})

	if err != nil {
		return nil, err
	}

	if err := conn.Ping(context.Background()); err != nil {
		if exception, ok := err.(*clickhouse.Exception); ok {
			fmt.Printf("Exception [%d] %s \n%s\n", exception.Code, exception.Message, exception.StackTrace)
		}
		return nil, err
	}

	return conn, nil
}
