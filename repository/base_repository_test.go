package repository

import (
	"context"
	"fmt"
	"github.com/hub1989/mongo-data/base_entity"
	"github.com/hub1989/mongo-data/configuration"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"testing"
)

type TestEntity struct {
	Id   primitive.ObjectID `bson:"_id" json:"id"`
	Name string             `bson:"name" json:"name"`
}

func (t TestEntity) GetId() primitive.ObjectID {
	return t.Id
}

func (t TestEntity) SetId(id primitive.ObjectID) {
	t.Id = id
}

type EntityTestSuite struct {
	suite.Suite
	MongoURI string
	MongoRepository[TestEntity]
	*mongo.Collection
}

func (s *EntityTestSuite) TearDownTest() {
	_, err := s.Collection.DeleteMany(context.Background(), bson.M{})
	if err != nil {
		log.Error(err)
	}
}

func (s *EntityTestSuite) SetupSuite() {
	port := "27017/tcp"
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "mongo:6",
		ExposedPorts: []string{port},
		Env: map[string]string{
			"MONGO_INITDB_ROOT_USERNAME": "test",
			"MONGO_INITDB_ROOT_PASSWORD": "test",
			"MONGO_INITDB_DATABASE":      "admin",
		},
	}

	mongoC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})

	if err != nil {
		log.Fatal(err)
	}

	endpoint, _ := mongoC.Endpoint(ctx, "")
	s.MongoURI = fmt.Sprintf("mongodb://test:test@%s/", endpoint)

	configService := configuration.DefaultDBConfigService{
		MongoURI:     s.MongoURI,
		DatabaseName: "test-db",
	}
	client := configService.ConnectDB()

	collection := client.Database(configService.DatabaseName).Collection("test_entities")
	s.Collection = collection
	repository := MongoRepository[TestEntity]{Collection: collection}
	s.MongoRepository = repository
}

func (s *EntityTestSuite) TestMongoRepository_Create() {
	request := TestEntity{
		Id:   primitive.NewObjectID(),
		Name: "test",
	}

	saved, err := s.MongoRepository.Save(context.Background(), request)
	s.Nil(err)
	s.NotNil(saved)

	fromDB, err := s.MongoRepository.FindById(context.TODO(), request.Id)
	s.Nil(err)

	s.Equal(request.Name, fromDB.Name)
}

func (s *EntityTestSuite) TestMongoRepository_SaveMany() {
	request1 := TestEntity{
		Id:   primitive.NewObjectID(),
		Name: "test 1",
	}

	request2 := TestEntity{
		Id:   primitive.NewObjectID(),
		Name: "test 2",
	}
	var entities []interface{}
	entities = append(entities, request1)
	entities = append(entities, request2)

	resp, err := s.MongoRepository.SaveMany(context.Background(), entities)
	s.Nil(err)

	s.Equal(2, len(resp))

	noOfDocsInDB, err := s.MongoRepository.CountDocumentsInCollected(context.TODO())
	s.Nil(err)

	s.Equal(int64(2), noOfDocsInDB)

	inDB, err := s.MongoRepository.FindByIds(context.TODO(), []primitive.ObjectID{request1.Id, request2.Id})
	s.Nil(err)

	s.Equal(2, len(inDB))
}

func (s *EntityTestSuite) TestMongoRepository_Update() {
	request := TestEntity{
		Id:   primitive.NewObjectID(),
		Name: "test",
	}

	saved, err := s.MongoRepository.Save(context.Background(), request)
	s.Nil(err)
	s.NotNil(saved)

	saved.Name = "Updated"
	updated, err := s.MongoRepository.Update(context.TODO(), *saved)
	s.Nil(err)
	s.NotNil(updated)
	s.Equal(saved.Name, updated.Name)
}

func (s *EntityTestSuite) TestMongoRepository_UpdateMany() {
	request := TestEntity{
		Id:   primitive.NewObjectID(),
		Name: "test",
	}

	saved, err := s.MongoRepository.Save(context.Background(), request)
	s.Nil(err)
	s.NotNil(saved)

	saved.Name = "Updated"
	updated, err := s.MongoRepository.UpdateMany(context.TODO(), []TestEntity{*saved})
	s.Nil(err)
	s.NotNil(updated)
	s.Equal(1, len(updated))
	s.Equal(saved.Name, updated[0].Name)
}

func (s *EntityTestSuite) TestMongoRepository_FindById() {
	request := TestEntity{
		Id:   primitive.NewObjectID(),
		Name: "test",
	}

	saved, err := s.MongoRepository.Save(context.Background(), request)
	s.Nil(err)
	s.NotNil(saved)

	ctx := context.Background()

	entityFound, err := s.MongoRepository.FindById(ctx, request.Id)
	s.Nil(err)
	assert.Equal(s.T(), request.Id, entityFound.Id)
}

func (s *EntityTestSuite) TestMongoRepository_FindByIds() {
	request1 := TestEntity{
		Id:   primitive.NewObjectID(),
		Name: "test 1",
	}

	request2 := TestEntity{
		Id:   primitive.NewObjectID(),
		Name: "test 2",
	}
	var entities []interface{}
	entities = append(entities, request1)
	entities = append(entities, request2)

	_, err := s.MongoRepository.SaveMany(context.Background(), entities)
	s.Nil(err)

	inDB, err := s.MongoRepository.FindByIds(context.TODO(), []primitive.ObjectID{request1.Id, request2.Id})
	s.Nil(err)

	s.Equal(2, len(inDB))
}

func (s *EntityTestSuite) TestMongoRepository_Delete() {
	id := primitive.NewObjectID()

	request := TestEntity{
		Id:   id,
		Name: "test",
	}

	saved, err := s.MongoRepository.Save(context.Background(), request)
	s.Nil(err)
	s.NotNil(saved)

	ctx := context.Background()

	err = s.Delete(ctx, id)
	s.Nil(err)

	_, err = s.MongoRepository.FindById(ctx, id)
	s.NotNil(err)
}

func (s *EntityTestSuite) TestMongoRepository_DeleteMany() {
	id := primitive.NewObjectID()

	request := TestEntity{
		Id:   id,
		Name: "test",
	}

	saved, err := s.MongoRepository.Save(context.Background(), request)
	s.Nil(err)
	s.NotNil(saved)

	ctx := context.Background()

	err = s.MongoRepository.DeleteMany(ctx, []primitive.ObjectID{id})
	s.Nil(err)

	_, err = s.MongoRepository.FindById(ctx, id)
	s.NotNil(err)
}

func (s *EntityTestSuite) TestMongoRepository_FindAllByPageable() {
	request1 := TestEntity{
		Id:   primitive.NewObjectID(),
		Name: "test 1",
	}

	request2 := TestEntity{
		Id:   primitive.NewObjectID(),
		Name: "test 2",
	}
	var entities []interface{}
	entities = append(entities, request1)
	entities = append(entities, request2)

	_, err := s.MongoRepository.SaveMany(context.Background(), entities)
	s.Nil(err)

	ctx := context.Background()

	page1, err := s.MongoRepository.FindAllPageable(base_entity.PageableDBRequest{
		NumberPerPage: 1,
	}, ctx)

	s.Nil(err)
	s.Equal(int64(1), page1.NoOfItemsInBatch)

	page2, err := s.MongoRepository.FindAllPageable(base_entity.PageableDBRequest{
		NumberPerPage: 1,
		LastItemId:    page1.LastItemId,
	}, ctx)

	s.Nil(err)
	s.Equal(int64(1), page2.NoOfItemsInBatch)
	s.NotEqual(page1.Data[0].Id.Hex(), page2.Data[0].Id.Hex())
}

func TestDeliveryTestSuite(t *testing.T) {
	suite.Run(t, new(EntityTestSuite))
}
