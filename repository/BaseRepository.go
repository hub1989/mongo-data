package repository

import (
	"context"
	"fmt"
	"github.com/hub1989/mongo-data/base_entity"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Repository[T base_entity.Entity] interface {
	Save(ctx context.Context, entity T) (*T, error)
	SaveMany(ctx context.Context, entities []interface{}) ([]string, error)
	Update(ctx context.Context, entity T) (*T, error)
	UpdateMany(ctx context.Context, entities []T) ([]*T, error)
	FindById(ctx context.Context, id primitive.ObjectID) (*T, error)
	Delete(ctx context.Context, id primitive.ObjectID) error
	DeleteMany(ctx context.Context, ids []primitive.ObjectID) error
	FindByIds(ctx context.Context, ids []primitive.ObjectID) ([]*T, error)
	FindEntityDocumentsByFilter(ctx context.Context, filter bson.M, opts ...*options.FindOptions) ([]*T, error)
	FindEntityDocumentByFilter(ctx context.Context, filter bson.M) (*T, error)
	CountDocumentsInCollected(ctx context.Context) (int64, error)
	FindAllPageable(request base_entity.PageableDBRequest, ctx context.Context) (*base_entity.PageableDBResponse[T], error)
}

type MongoRepository[T base_entity.Entity] struct {
	Collection *mongo.Collection
}

func (p MongoRepository[T]) Save(ctx context.Context, entity T) (*T, error) {
	res, err := p.Collection.InsertOne(ctx, entity)

	if err != nil {
		log.WithError(err).Error(fmt.Sprintf("could not save %s collected entity", p.Collection.Name()))
		return nil, err
	}

	log.WithFields(log.Fields{
		"id": res.InsertedID,
	}).Info(fmt.Sprintf("saved %s entity", p.Collection.Name()))

	return &entity, nil
}

func (p MongoRepository[T]) SaveMany(ctx context.Context, entities []interface{}) ([]string, error) {
	res, err := p.Collection.InsertMany(ctx, entities)

	if err != nil {
		log.WithError(err).Error(fmt.Sprintf("could not save %s collected entity", p.Collection.Name()))
		return nil, err
	}

	var ids []string
	for _, objId := range res.InsertedIDs {
		id := objId.(primitive.ObjectID)
		ids = append(ids, id.Hex())
	}

	log.WithFields(log.Fields{
		"count": len(ids),
	}).Info(fmt.Sprintf("saved %s entities", p.Collection.Name()))

	return ids, nil
}

func (p MongoRepository[T]) Update(ctx context.Context, entity T) (*T, error) {

	idFilter := bson.M{
		"_id": entity.GetId(),
	}

	updateFilter := bson.M{
		"$set": entity,
	}

	opts := options.Update().SetUpsert(true)

	result, err := p.Collection.UpdateOne(ctx, idFilter, updateFilter, opts)

	if err != nil {
		log.WithError(err).Error("could not update entity: ", result.UpsertedID)
		return nil, err
	}
	return &entity, nil
}

func (p MongoRepository[T]) UpdateMany(ctx context.Context, entities []T) ([]*T, error) {
	var result []*T
	for _, entity := range entities {
		res, err := p.Update(ctx, entity)
		if err != nil {
			return nil, err
		}

		result = append(result, res)
	}

	return result, nil
}

func (p MongoRepository[T]) FindById(ctx context.Context, id primitive.ObjectID) (*T, error) {
	filter := bson.M{"_id": id}
	return p.FindEntityDocumentByFilter(ctx, filter)
}

func (p MongoRepository[T]) Delete(ctx context.Context, id primitive.ObjectID) error {
	return p.DeleteMany(ctx, []primitive.ObjectID{id})
}

func (p MongoRepository[T]) DeleteMany(ctx context.Context, ids []primitive.ObjectID) error {
	filter := bson.M{
		"_id": bson.M{
			"$in": ids,
		},
	}

	res, err := p.Collection.DeleteMany(ctx, filter)

	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"response": res.DeletedCount,
	}).Info(fmt.Sprintf("deleted %s entities", p.Collection.Name()))

	return nil
}

func (p MongoRepository[T]) FindByIds(ctx context.Context, ids []primitive.ObjectID) ([]*T, error) {
	filter := bson.M{
		"_id": bson.M{
			"$in": ids,
		},
	}

	return p.FindEntityDocumentsByFilter(ctx, filter)
}

func (p MongoRepository[T]) FindEntityDocumentsByFilter(ctx context.Context, filter bson.M, opts ...*options.FindOptions) ([]*T, error) {
	var records []*T

	reslts, err := p.Collection.Find(ctx, filter, opts...)
	if err != nil {
		return nil, err
	}

	return p.handleResultCursorForPointer(reslts, ctx, records)
}

func (p MongoRepository[T]) FindEntityDocumentsByFilterForObject(ctx context.Context, filter bson.M, opts ...*options.FindOptions) ([]T, error) {
	var records []T

	reslts, err := p.Collection.Find(ctx, filter, opts...)
	if err != nil {
		return nil, err
	}

	return p.handleResultCursorForObject(reslts, ctx, records)
}

func (p MongoRepository[T]) FindEntityDocumentByFilter(ctx context.Context, filter bson.M) (*T, error) {
	var responseType T
	err := p.Collection.FindOne(ctx, filter).Decode(&responseType)
	if err != nil {
		return nil, err
	}
	return &responseType, nil
}

func (p MongoRepository[T]) CountDocumentsInCollected(ctx context.Context) (int64, error) {
	documents, err := p.Collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return 0, err
	}

	return documents, nil
}

func (p MongoRepository[T]) FindAllPageable(request base_entity.PageableDBRequest, ctx context.Context) (*base_entity.PageableDBResponse[T], error) {
	filter := bson.M{}

	noOfDocuments, err := p.Collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, err
	}

	if request.LastItemId != "" {
		obj, err := primitive.ObjectIDFromHex(request.LastItemId)
		if err != nil {
			return nil, err
		}

		filter = bson.M{
			"_id": bson.M{
				"$lt": obj,
			},
		}
	}

	findOptions := options.FindOptions{
		Limit: &request.NumberPerPage,
		Sort: bson.M{
			"_id": -1,
		},
	}

	response, err := p.FindEntityDocumentsByFilterForObject(ctx, filter, &findOptions)
	if err != nil {
		return nil, err
	}

	var lastItemId string
	if len(response) > 0 {

		lastItemId = response[len(response)-1].GetId().Hex()
	}

	return &base_entity.PageableDBResponse[T]{
		Data:             response,
		NumberPerPage:    request.NumberPerPage,
		LastItemId:       lastItemId,
		Total:            noOfDocuments,
		NoOfItemsInBatch: int64(len(response)),
	}, nil
}

func (p MongoRepository[T]) handleResultCursorForPointer(records *mongo.Cursor, ctx context.Context, entities []*T) ([]*T, error) {
	defer records.Close(ctx)
	for records.Next(ctx) {
		var entity T
		if err := records.Decode(&entity); err != nil {
			log.WithError(err)
			return nil, err
		}
		entities = append(entities, &entity)
	}

	return entities, nil
}

func (p MongoRepository[T]) handleResultCursorForObject(records *mongo.Cursor, ctx context.Context, entities []T) ([]T, error) {
	defer records.Close(ctx)
	for records.Next(ctx) {
		var entity T
		if err := records.Decode(&entity); err != nil {
			log.WithError(err)
			return nil, err
		}
		entities = append(entities, entity)
	}

	return entities, nil
}
