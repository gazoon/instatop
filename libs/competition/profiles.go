package competition

import (
	"github.com/gazoon/go-utils"
	"github.com/gazoon/go-utils/mongo"
	"instarate/libs/instagram"

	"context"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
	"time"
)

const (
	ProfilePhotosFolder = "profile_photos/"
)

var (
	ProfileExistsErr = errors.New("profile already exists")
)

type InstProfile struct {
	Username      string    `bson:"username"`
	PhotoPath     string    `bson:"photo"`
	PhotoInstCode string    `bson:"photo_code"`
	Followers     int       `bson:"followers"`
	AddedAt       time.Time `bson:"added_at"`
}

func NewProfile(username, photoInstCode string, followers int) *InstProfile {
	addedAt := utils.UTCNow()
	photoStoragePath := ProfilePhotosFolder + username + "-" + uuid.NewV4().String()
	return &InstProfile{
		Username: username, PhotoPath: photoStoragePath,
		PhotoInstCode: photoInstCode, Followers: followers, AddedAt: addedAt,
	}
}

func (self *InstProfile) GetProfileLink() string {
	return instagram.BuildProfileUrl(self.Username)
}

func (self InstProfile) String() string {
	return utils.ObjToString(&self)
}

type ProfilesStorage struct {
	client *mgo.Collection
}

func NewProfilesStorage(mongoSettings *utils.MongoDBSettings) (*ProfilesStorage, error) {
	collection, err := mongo.ConnectCollection(mongoSettings)
	if err != nil {
		return nil, err
	}
	return &ProfilesStorage{collection}, nil
}

func (self *ProfilesStorage) Create(ctx context.Context, model *InstProfile) error {
	err := self.client.Insert(model)
	if mgo.IsDup(err) {
		return ProfileExistsErr
	}
	return errors.Wrap(err, "insert new profile document")
}

func (self *ProfilesStorage) Get(ctx context.Context, username string) (*InstProfile, error) {
	result := &InstProfile{}
	err := self.client.Find(bson.M{"username": username}).One(result)
	if err != nil {
		return nil, errors.Wrap(err, "get single profile document")
	}
	return result, nil
}

func (self *ProfilesStorage) Save(ctx context.Context, model *InstProfile) error {
	err := self.client.Update(bson.M{"username": model.Username}, model)
	if err != nil {
		return errors.Wrap(err, "update single profile document")
	}
	return nil
}

func (self *ProfilesStorage) GetAll(ctx context.Context) ([]*InstProfile, error) {
	var result []*InstProfile
	err := self.client.Find(nil).All(&result)
	if err != nil {
		return nil, errors.Wrap(err, "get all profile document")
	}
	return result, nil
}

func (self *ProfilesStorage) GetMultiple(ctx context.Context, usernames []string) ([]*InstProfile, error) {
	var result []*InstProfile
	err := self.client.Find(bson.M{"username": bson.M{"$in": usernames}}).All(&result)
	if err != nil {
		return nil, errors.Wrap(err, "get multiple profiles documents")
	}
	return result, nil
}

func (self *ProfilesStorage) Delete(ctx context.Context, usernames []string) error {
	_, err := self.client.RemoveAll(bson.M{"username": bson.M{"$in": usernames}})
	return errors.Wrap(err, "delete all profile documents by usernames")
}

func (self *ProfilesStorage) CreateIndexes() error {
	var err error

	err = self.client.EnsureIndex(mgo.Index{Key: []string{"username"}, Unique: true})
	if err != nil {
		return errors.Wrap(err, "unique key: username")
	}

	return nil
}
