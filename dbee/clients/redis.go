package clients

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kndndrj/nvim-dbee/dbee/clients/common"
	"github.com/kndndrj/nvim-dbee/dbee/conn"
	"github.com/kndndrj/nvim-dbee/dbee/models"
	"github.com/redis/go-redis/v9"
)

// Register client
func init() {
	c := func(url string) (conn.Client, error) {
		return NewRedis(url)
	}
	_ = Store.Register("redis", c)
}

type RedisClient struct {
	redis *redis.Client
}

func NewRedis(url string) (*RedisClient, error) {
	c := redis.NewClient(&redis.Options{
		Addr:     url,
		Password: "",
		DB:       0,
	})

	return &RedisClient{
		redis: c,
	}, nil
}

func (c *RedisClient) Query(query string) (models.IterResult, error) {
	cmd, err := parseRedisCmd(query)
	if err != nil {
		return nil, err
	}

	resp, err := c.redis.Do(context.Background(), cmd...).Result()
	if err != nil {
		return nil, err
	}

	// parse response
	var rows []models.Row
	switch rpl := resp.(type) {
	case int64:
		rows = []models.Row{{rpl}}
	case string:
		rows = []models.Row{{rpl}}
	case []any:
		rows = sliceToRows(rpl, -1)
	case map[any]any:
		for k, v := range rpl {
			rows = append(rows, models.Row{k, v})
		}
	case nil:
		return nil, errors.New("no reponse from redis")
	default:
		return nil, fmt.Errorf("unknown type reponse from redis: %T", rpl)
	}

	// build result
	max := len(rows) - 1
	i := 0
	result := common.NewResultBuilder().
		WithNextFunc(func() (models.Row, error) {
			if i > max {
				return nil, nil
			}
			val := rows[i]
			i++
			return val, nil
		}).
		WithHeader(models.Header{"Reply"}).
		WithMeta(models.Meta{
			Query:     query,
			Timestamp: time.Now(),
		}).
		Build()

	return result, err
}

func (c *RedisClient) Layout() ([]models.Layout, error) {
	return []models.Layout{
		{
			Name:     "DB",
			Schema:   "",
			Database: "",
			Type:     models.LayoutTable,
		},
	}, nil
}

func (c *RedisClient) Close() {
	c.redis.Close()
}

// ErrUnmatchedDoubleQuote and ErrUnmatchedSingleQuote are errors returned from ParseRedisCmd
var (
	ErrUnmatchedDoubleQuote = func(position int) error { return fmt.Errorf("syntax error: unmatched double quote at: %d", position) }
	ErrUnmatchedSingleQuote = func(position int) error { return fmt.Errorf("syntax error: unmatched single quote at: %d", position) }
)

// parseRedisCmd parses string command into args for redis.Do
func parseRedisCmd(unparsed string) ([]any, error) {
	// error helper
	quoteErr := func(quote rune, position int) error {
		if quote == '"' {
			return ErrUnmatchedDoubleQuote(position)
		} else {
			return ErrUnmatchedSingleQuote(position)
		}
	}

	// return array
	var fields []any
	// what char is the current quote
	var blank rune
	var currentQuote struct {
		char     rune
		position int
	}
	// is the current char escaped or not?
	var escaped bool

	sb := &strings.Builder{}
	for i, r := range unparsed {
		// handle unescaped quotes
		if !escaped && (r == '"' || r == '\'') {
			// next char
			next := byte(' ')
			if i < len(unparsed)-1 {
				next = unparsed[i+1]
			}

			if r == currentQuote.char {
				if next != ' ' {
					return nil, quoteErr(r, i+1)
				}
				// end quote
				currentQuote.char = blank
				continue
			} else if currentQuote.char == blank {
				// start quote
				currentQuote.char = r
				currentQuote.position = i + 1
				continue
			}
		}

		// handle escapes
		if r == '\\' {
			escaped = true
			continue
		}

		// handle word end
		if currentQuote.char == blank && r == ' ' {
			fields = append(fields, sb.String())
			sb.Reset()
			continue
		}

		escaped = false
		sb.WriteRune(r)
	}

	// check if quote is not closed
	if currentQuote.char != blank {
		return nil, quoteErr(currentQuote.char, currentQuote.position)
	}

	// write last word
	if sb.Len() > 0 {
		fields = append(fields, sb.String())
	}

	return fields, nil
}

// sliceToRows expands []any slice and any possible nested slices to multiple rows
func sliceToRows(slice []any, level int) []models.Row {
	var rows []models.Row

	var prefix []any
	for i := 0; i < level; i++ {
		prefix = append(prefix, "")
	}

	for _, v := range slice {
		if nested, ok := v.([]any); ok {
			rs := sliceToRows(nested, level+1)
			rows = append(rows, rs...)
		} else {
			row := append(prefix, v)
			rows = append(rows, row)
		}
	}
	return rows
}
