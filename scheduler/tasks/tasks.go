package tasks

import (
	"context"
	"github.com/gazoon/go-utils"
	"github.com/gazoon/go-utils/mongo"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"
	"time"
)

type Task struct {
	ChatId int                    `bson:"chat_id" mapstructure:"chat_id"`
	Name   string                 `bson:"name" mapstructure:"name"`
	Args   map[string]interface{} `bson:"args" mapstructure:"args"`
	DoAt   time.Time              `bson:"do_at" mapstructure:"do_at"`
}

func NewTaskWithoutArgs(name string, chatId int, doAt time.Time) *Task {
	return NewTask(name, chatId, doAt, nil)
}

func NewTask(name string, chatId int, doAt time.Time, args map[string]interface{}) *Task {
	return &Task{chatId, name, args, doAt}
}

func (self Task) String() string {
	return utils.ObjToString(&self)
}

var (
	TaskAlreadyExistsErr = errors.New(
		"Task with that name already exists for the chat")
)

type Storage struct {
	client *mgo.Collection
}

func NewStorage(mongoSettings *utils.MongoDBSettings) (*Storage, error) {
	collection, err := mongo.ConnectCollection(mongoSettings)
	if err != nil {
		return nil, err
	}
	return &Storage{collection}, nil
}

func (self *Storage) GetAndRemoveTask(ctx context.Context) (*Task, error) {
	currentTime := utils.UTCNow()
	task := &Task{}
	_, err := self.client.Find(bson.M{"do_at": bson.M{"$lte": currentTime}}).
		Apply(mgo.Change{Remove: true}, task)
	if err == mgo.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "get and remove ready task document")
	}
	return task, nil
}

func (self *Storage) CreateTask(ctx context.Context, task *Task) error {
	err := self.client.Insert(task)
	if mgo.IsDup(err) {
		return TaskAlreadyExistsErr
	}
	return errors.Wrap(err, "insert new task document")
}

func (self *Storage) CreateOrReplaceTask(ctx context.Context, task *Task) error {
	_, err := self.client.Upsert(bson.M{"chat_id": task.ChatId, "name": task.Name}, task)
	return errors.Wrap(err, "upsert task document")
}

func (self *Storage) DeleteTask(ctx context.Context, chatId int, name string) error {
	err := self.client.Remove(bson.M{"chat_id": chatId, "name": name})
	return errors.Wrap(err, "delete task document")
}
