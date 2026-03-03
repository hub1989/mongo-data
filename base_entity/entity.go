package base_entity

import "go.mongodb.org/mongo-driver/v2/bson"

type Entity interface {
	GetId() bson.ObjectID
	SetId(id bson.ObjectID)
}
