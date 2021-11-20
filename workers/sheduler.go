package workers

import (
	"fmt"
	"github.com/RichardKnop/machinery/v1"
	"github.com/RichardKnop/machinery/v1/config"
	"github.com/RichardKnop/machinery/v1/log"
	"github.com/RichardKnop/machinery/v1/tasks"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"netwitter/schemas"
	"time"
)

type Scheduler struct {
	server *machinery.Server
}

func NewScheduler(brokerUrl string) *Scheduler {
	cfg := &config.Config{
		DefaultQueue:    "tasks",
		ResultsExpireIn: int(time.Hour.Seconds()),
		Broker:          fmt.Sprintf("redis://%s", brokerUrl),
		ResultBackend:   fmt.Sprintf("redis://%s", brokerUrl),
		Redis: &config.RedisConfig{
			MaxIdle:                3,
			IdleTimeout:            240,
			ReadTimeout:            15,
			WriteTimeout:           15,
			ConnectTimeout:         15,
			NormalTasksPollPeriod:  1000,
			DelayedTasksPollPeriod: 500,
		},
	}

	server, err := machinery.NewServer(cfg)
	if err != nil {
		panic(err)
	}

	scheduler := &Scheduler{
		server: server,
	}
	return scheduler
}

func (sh *Scheduler) Listen() error {
	worker := sh.server.NewWorker("worker", 0)
	errorHandler := func(err error) {
		log.ERROR.Println("Something went wrong: ", err)
	}
	worker.SetErrorHandler(errorHandler)

	return worker.Launch()
}

func (sh *Scheduler) PublishSpreadPostOverSubs(userId schemas.UserId, postId schemas.PostId) error {
	task := &tasks.Signature{
		Name: "SpreadPostOverSubscribers",
		Args: []tasks.Arg{
			{
				Type:  "string",
				Value: string(userId),
			},
			{
				Type:  "string",
				Value: primitive.ObjectID(postId).Hex(),
			},
		},
	}
	_, err := sh.server.SendTask(task)
	return err
}

func (sh *Scheduler) PublishCollectPostsToPersonalFeed(userId schemas.UserId, from schemas.UserId) error {
	task := &tasks.Signature{
		Name: "CollectPostsToPersonalFeed",
		Args: []tasks.Arg{
			{
				Type:  "string",
				Value: string(userId),
			},
			{
				Type:  "string",
				Value: string(from),
			},
		},
	}
	_, err := sh.server.SendTask(task)
	return err
}

func (sh *Scheduler) Register(executor PostsTasksExecutor) error {
	return sh.server.RegisterTasks(executor.GetCommandsMapping())
}
