/*
 * Copyright 2024 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package mongo

import (
	"context"
	"errors"
	"github.com/SENERGY-Platform/permissions-v2/pkg/configuration"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"reflect"
	"runtime/debug"
	"strings"
	"time"
)

type Database struct {
	config configuration.Config
	client *mongo.Client
}

var CreateCollections = []func(db *Database) error{}

func New(conf configuration.Config) (*Database, error) {
	ctx, _ := getTimeoutContext()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(conf.MongoUrl))
	if err != nil {
		return nil, err
	}
	db := &Database{config: conf, client: client}
	for _, creators := range CreateCollections {
		err = creators(db)
		if err != nil {
			client.Disconnect(context.Background())
			return nil, err
		}
	}
	if conf.MigrateFromMongoUrl != "" && conf.MigrateFromMongoUrl != "-" {
		topics, err := db.ListTopics(context.Background(), model.ListOptions{})
		if err != nil {
			return nil, err
		}
		if len(topics) == 0 {
			err = MigrateDb(db, conf)
			if err != nil {
				return nil, err
			}
		}
	}
	return db, nil
}

func MigrateDb(db *Database, origConf configuration.Config) error {
	log.Println("Migrating database...")
	ctx, _ := getTimeoutContext()
	conf := origConf
	conf.MongoUrl = origConf.MigrateFromMongoUrl

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(conf.MongoUrl))
	if err != nil {
		return err
	}

	source := &Database{config: conf, client: client}

	topics, err := source.ListTopics(context.Background(), model.ListOptions{})
	if err != nil {
		return err
	}
	for _, topic := range topics {
		log.Printf("migrate topic %s\n", topic.Id)
		ctx, _ := getTimeoutContext()
		err = db.SetTopic(ctx, topic)
		if err != nil {
			return err
		}
	}

	cursor, err := source.permissionsCollection().Find(context.Background(), bson.M{})
	if err != nil {
		debug.PrintStack()
		return err
	}
	defer cursor.Close(context.Background())
	for cursor.Next(context.Background()) {
		element := PermissionsEntry{}
		err = cursor.Decode(&element)
		if err != nil {
			debug.PrintStack()
			return err
		}
		log.Printf("migrate permission entry %s %s\n", element.TopicId, element.Id)
		ctx, _ := getTimeoutContext()
		_, err = db.permissionsCollection().ReplaceOne(ctx, bson.M{PermissionsEntryBson.TopicId: element.TopicId, PermissionsEntryBson.Id: element.Id}, element, options.Replace().SetUpsert(true))
		if err != nil {
			debug.PrintStack()
			return err
		}
	}
	err = cursor.Err()
	if err != nil {
		debug.PrintStack()
		return err
	}
	return nil
}

func (this *Database) CreateId() string {
	return uuid.NewString()
}

func (this *Database) ensureIndex(collection *mongo.Collection, indexname string, indexKey string, asc bool, unique bool) error {
	ctx, _ := getTimeoutContext()
	var direction int32 = -1
	if asc {
		direction = 1
	}
	_, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{indexKey, direction}},
		Options: options.Index().SetName(indexname).SetUnique(unique),
	})
	if err != nil {
		debug.PrintStack()
	}
	return err
}

func (this *Database) ensureCompoundIndex(collection *mongo.Collection, indexname string, asc bool, unique bool, indexKeys ...string) error {
	ctx, _ := getTimeoutContext()
	var direction int32 = -1
	if asc {
		direction = 1
	}
	keys := []bson.E{}
	for _, key := range indexKeys {
		keys = append(keys, bson.E{Key: key, Value: direction})
	}
	_, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D(keys),
		Options: options.Index().SetName(indexname).SetUnique(unique),
	})
	return err
}

func (this *Database) removeIndex(collection *mongo.Collection, indexname string) error {
	_, err := collection.Indexes().DropOne(context.Background(), indexname)
	if err != nil {
		if strings.Contains(err.Error(), "IndexNotFound") {
			return nil
		} else {
			debug.PrintStack()
			return err
		}
	}
	return nil
}

func (this *Database) Disconnect() {
	timeout, _ := context.WithTimeout(context.Background(), 10*time.Second)
	log.Println(this.client.Disconnect(timeout))
}

func getBsonFieldName(obj interface{}, fieldName string) (bsonName string, err error) {
	field, found := reflect.TypeOf(obj).FieldByName(fieldName)
	if !found {
		return "", errors.New("field '" + fieldName + "' not found")
	}
	tags, err := bsoncodec.DefaultStructTagParser.ParseStructTags(field)
	return tags.Name, err
}

func getTimeoutContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Second)
}

func getBsonFieldObject[T any]() T {
	v := new(T)
	err := fillObjectWithItsBsonFieldNames(v, nil)
	if err != nil {
		panic(err)
	}
	return *v
}

func fillObjectWithItsBsonFieldNames(ptr interface{}, prefix []string) error {
	ptrval := reflect.ValueOf(ptr)
	objval := reflect.Indirect(ptrval)
	objecttype := objval.Type()
	for i := 0; i < objecttype.NumField(); i++ {
		field := objecttype.Field(i)
		if field.Type.Kind() == reflect.String {
			tags, err := bsoncodec.DefaultStructTagParser.ParseStructTags(field)
			if err != nil {
				return err
			}
			objval.Field(i).SetString(strings.Join(append(prefix, tags.Name), "."))
		}
		if field.Type.Kind() == reflect.Slice && field.Type.Elem().Kind() == reflect.String {
			tags, err := bsoncodec.DefaultStructTagParser.ParseStructTags(field)
			if err != nil {
				return err
			}
			objval.Field(i).Set(reflect.ValueOf([]string{strings.Join(append(prefix, tags.Name), ".")}))
		}
		if field.Type.Kind() == reflect.Struct {
			tags, err := bsoncodec.DefaultStructTagParser.ParseStructTags(field)
			if err != nil {
				return err
			}
			if tags.Inline {
				err = fillObjectWithItsBsonFieldNames(objval.Field(i).Addr().Interface(), prefix)
			} else {
				err = fillObjectWithItsBsonFieldNames(objval.Field(i).Addr().Interface(), append(prefix, tags.Name))
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}
