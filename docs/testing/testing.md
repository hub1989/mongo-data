---
tags:
 - mongo
 - testing
 - test containers
---

## Test Dependencies
- test containers
- testify

## Test flow
- define an entity/document type you'd like to test for
```go
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
```

- set up a test suite using test containers
```go
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
```
- test the methods you're interested in
```go
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
```