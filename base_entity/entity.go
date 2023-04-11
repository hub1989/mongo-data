package base_entity

import "go.mongodb.org/mongo-driver/bson/primitive"

type Entity interface {
	GetId() primitive.ObjectID
	SetId(id primitive.ObjectID)
}
