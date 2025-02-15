package clients

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/kndndrj/nvim-dbee/dbee/clients/common"
	"github.com/kndndrj/nvim-dbee/dbee/conn"
	"github.com/kndndrj/nvim-dbee/dbee/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Register client
func init() {
	c := func(url string) (conn.Client, error) {
		return NewMongo(url)
	}
	_ = Store.Register("mongo", c)
}

func getDatabaseName(url string) (string, error) {
	r, err := regexp.Compile(`mongo.*//(.*:[0-9]+,?)+/(?P<dbname>.*?)(\?|$)`)
	if err != nil {
		return "", err
	}

	// get submatch index
	getSubmatchIndex := func(submatch []string, name string) (int, error) {
		for i, n := range submatch {
			if n == name {
				return i, nil
			}
		}
		return 0, errors.New("no submatch found")
	}
	i, err := getSubmatchIndex(r.SubexpNames(), "dbname")
	if err != nil {
		return "", err
	}

	// get database name from capture group (with index)
	submatch := r.FindStringSubmatch(url)
	if len(submatch) < 1 {
		return "", errors.New("url doesn't comply to schema")
	}
	dbName := submatch[i]
	if dbName == "" {
		return "", errors.New("no dbname found")
	}

	return dbName, nil
}

type MongoClient struct {
	c      *mongo.Client
	dbName string
}

func NewMongo(url string) (*MongoClient, error) {
	// get database name from url
	dbName, err := getDatabaseName(url)
	if err != nil {
		return nil, fmt.Errorf("mongo: invalid url: %v", err)
	}

	opts := options.Client().ApplyURI(url)
	client, err := mongo.Connect(context.TODO(), opts)
	if err != nil {
		return nil, err
	}

	return &MongoClient{
		c:      client,
		dbName: dbName,
	}, nil
}

func (c *MongoClient) Query(query string) (models.IterResult, error) {
	db := c.c.Database(c.dbName)

	var command any
	err := bson.UnmarshalExtJSON([]byte(query), false, &command)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal command: \"%v\" to bson: %v", query, err)
	}

	var resp bson.M
	err = db.RunCommand(context.TODO(), command).Decode(&resp)
	if err != nil {
		return nil, err
	}

	// check if cursor exists and create an appropriate func
	var nextFn func() (models.Row, error)

	cur, ok := resp["cursor"]
	if ok {
		c := make(chan any)

		cursor := cur.(bson.M)
		if !ok {
			return nil, errors.New("type assertion for cursor object failed")
		}

		go func() {
			defer close(c)
			for _, b := range cursor {
				batch, ok := b.(bson.A)
				if !ok {
					continue
				}
				for _, item := range batch {
					c <- item
				}
			}
		}()

		nextFn = func() (models.Row, error) {
			val, ok := <-c
			if !ok {
				return nil, nil
			}
			parsed, err := json.MarshalIndent(val, "", "  ")
			if err != nil {
				return nil, err
			}
			return models.Row{string(parsed)}, nil
		}
	} else {
		once := false
		nextFn = func() (models.Row, error) {
			if !once {
				once = true
				parsed, err := json.MarshalIndent(resp, "", "  ")
				if err != nil {
					return nil, err
				}
				return models.Row{string(parsed)}, nil
			}
			return nil, nil
		}

	}

	// build result
	result := common.NewResultBuilder().
		WithNextFunc(nextFn).
		WithHeader(models.Header{"Reply"}).
		WithMeta(models.Meta{
			Query:     query,
			Timestamp: time.Now(),
		}).
		Build()

	return result, nil
}

func (c *MongoClient) Layout() ([]models.Layout, error) {
	collections, err := c.c.Database(c.dbName).ListCollectionNames(context.TODO(), bson.D{})
	if err != nil {
		return nil, err
	}

	var layout []models.Layout

	for _, coll := range collections {
		layout = append(layout, models.Layout{
			Name:     coll,
			Schema:   "",
			Database: "",
			Type:     models.LayoutTable,
		})
	}

	return layout, nil
}

func (c *MongoClient) Close() {
	_ = c.c.Disconnect(context.TODO())
}
