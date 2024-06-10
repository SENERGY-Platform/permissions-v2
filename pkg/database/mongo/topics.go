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
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

var TopicBson = getBsonFieldObject[model.Topic]()

func init() {
	CreateCollections = append(CreateCollections, func(db *Database) error {
		var err error
		collection := db.client.Database(db.config.MongoDatabase).Collection(db.config.MongoPermissionsCollection)
		err = db.ensureIndex(collection, "topicbyid", TopicBson.Id, true, true)
		if err != nil {
			return err
		}
		return nil
	})
}

func (this *Database) topicsCollection() *mongo.Collection {
	return this.client.Database(this.config.MongoDatabase).Collection(this.config.MongoTopicsCollection)
}

func (this *Database) SetTopic(ctx context.Context, topic model.Topic) error {
	if ctx == nil {
		ctx, _ = getTimeoutContext()
	}
	topic.LastUpdateUnixTimestamp = time.Now().UnixMilli()
	_, err := this.topicsCollection().ReplaceOne(ctx, bson.M{TopicBson.Id: topic.Id}, topic, options.Replace().SetUpsert(true))
	return err
}

func (this *Database) GetTopic(ctx context.Context, id string) (result model.Topic, exists bool, err error) {
	if ctx == nil {
		ctx, _ = getTimeoutContext()
	}
	temp := this.topicsCollection().FindOne(ctx, bson.M{TopicBson.Id: id})
	err = temp.Err()
	if err == mongo.ErrNoDocuments {
		return result, false, nil
	}
	if err != nil {
		return
	}
	err = temp.Decode(&result)
	if err == mongo.ErrNoDocuments {
		return result, false, nil
	}
	return result, true, err
}

func (this *Database) ListTopics(ctx context.Context, listOptions model.ListOptions) (result []model.Topic, err error) {
	if ctx == nil {
		ctx, _ = getTimeoutContext()
	}
	opt := options.Find()
	if listOptions.Limit > 0 {
		opt.SetLimit(listOptions.Limit)
	}
	if listOptions.Offset > 0 {
		opt.SetSkip(listOptions.Offset)
	}
	opt.SetSort(bson.D{{TopicBson.Id, 1}})
	cursor, err := this.topicsCollection().Find(ctx, bson.M{}, opt)
	if err != nil {
		return result, err
	}
	for cursor.Next(context.Background()) {
		element := model.Topic{}
		err = cursor.Decode(&element)
		if err != nil {
			return nil, err
		}
		result = append(result, element)
	}
	err = cursor.Err()
	return result, err
}

func (this *Database) DeleteTopic(ctx context.Context, id string) error {
	if ctx == nil {
		ctx, _ = getTimeoutContext()
	}
	_, err := this.topicsCollection().DeleteMany(ctx, bson.M{TopicBson.Id: id})
	if err != nil {
		return err
	}
	_, err = this.permissionsCollection().DeleteMany(ctx, bson.M{PermissionsEntryBson.TopicId: id})
	return err
}
