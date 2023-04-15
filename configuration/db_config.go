package configuration

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"
)

// DBConfigService configuration interface for
type DBConfigService interface {
	ConnectDB(mongoURI string) (*mongo.Client, error)
	GetCollection(client *mongo.Client, collectionName string) *mongo.Collection
	CreateIndexes(db *mongo.Client) error
}

// DefaultDBConfigService a default bare-bones configuration service.
// You can pass your own implementation
type DefaultDBConfigService struct {
	MongoURI     string
	DatabaseName string
}

// ConnectDB connect to database
// returns an error if the operation fails
func (d DefaultDBConfigService) ConnectDB() (*mongo.Client, error) {
	clientOptions := options.Client()
	clientOptions.ApplyURI(d.MongoURI)
	clientOptions.Monitor = otelmongo.NewMonitor()

	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		return nil, err
	}

	if err := client.Ping(context.Background(), readpref.Primary()); err != nil {
		return nil, err
	}

	log.Info("connected to DB: ", d.DatabaseName)
	return client, nil
}

// GetCollection get a mongo collection by collection name
func (d DefaultDBConfigService) GetCollection(client *mongo.Client, collectionName string) *mongo.Collection {
	collection := client.Database(d.DatabaseName).Collection(collectionName)
	return collection
}

// CreateIndexesForCollection create indexes for your collection
func (d DefaultDBConfigService) CreateIndexesForCollection(db *mongo.Client, collectionName string, indexes ...mongo.IndexModel) error {
	collection := d.GetCollection(db, collectionName)
	ctx := context.Background()

	_, err := collection.Indexes().CreateMany(ctx, indexes)

	if err != nil {
		log.WithError(err).Error(fmt.Sprintf("Failed to create indices for %s collection", collectionName))
		return err
	}

	return nil
}
