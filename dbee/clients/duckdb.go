//go:build cgo && ( (darwin && (amd64 || arm64)) || (linux && (amd64 || arm64 || riscv64)) )
package clients

import (
	"database/sql"
	"fmt"

	"github.com/kndndrj/nvim-dbee/dbee/clients/common"
	"github.com/kndndrj/nvim-dbee/dbee/conn"
	"github.com/kndndrj/nvim-dbee/dbee/models"
	_ "github.com/marcboeker/go-duckdb"
)

// Register client
func init() {
	c := func(url string) (conn.Client, error) {
		return NewDuck(url)
	}
	_ = Store.Register("duck", c)
}

type DuckClient struct {
	c *common.Client
}

func NewDuck(url string) (*DuckClient, error) {
	db, err := sql.Open("duckdb", url)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to duckdb database: %v", err)
	}

	return &DuckClient{
		c: common.NewClient(db),
	}, nil
}

func (c *DuckClient) Query(query string) (models.IterResult, error) {
	con, err := c.c.Conn()
	if err != nil {
		return nil, err
	}
	cb := func() {
		con.Close()
	}
	defer func() {
		if err != nil {
			cb()
		}
	}()

	rows, err := con.Query(query)
	if err != nil {
		return nil, err
	}
	rows.SetCallback(cb)
	return rows, nil
}

func (c *DuckClient) Layout() ([]models.Layout, error) {
	query := `SHOW TABLES;`

	rows, err := c.Query(query)
	if err != nil {
		return nil, err
	}

	var schema []models.Layout
	for {
		row, err := rows.Next()
		if row == nil {
			break
		}
		if err != nil {
			return nil, err
		}

		// We know for a fact there is only one string field (see query above)
		table := row[0].(string)
		schema = append(schema, models.Layout{
			Name:   table,
			Schema: "",
			// TODO:
			Database: "",
			Type:     models.LayoutTable,
		})
	}

	return schema, nil
}

func (c *DuckClient) Close() {
	c.c.Close()
}
