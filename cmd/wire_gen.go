// Code generated by Wire. DO NOT EDIT.

//go:generate wire
//+build !wireinject

package cmd

import (
	"github.com/jinzhu/gorm"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router"
	"github.com/traPtitech/traQ/service"
	"github.com/traPtitech/traQ/service/bot"
	"github.com/traPtitech/traQ/service/counter"
	"github.com/traPtitech/traQ/service/heartbeat"
	"github.com/traPtitech/traQ/service/imaging"
	"github.com/traPtitech/traQ/service/notification"
	"github.com/traPtitech/traQ/service/rbac"
	"github.com/traPtitech/traQ/service/sse"
	"github.com/traPtitech/traQ/service/viewer"
	"github.com/traPtitech/traQ/service/webrtcv3"
	"github.com/traPtitech/traQ/service/ws"
	"go.uber.org/zap"
)

import (
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "net/http/pprof"
)

// Injectors from serve_wire.go:

func newServer(hub2 *hub.Hub, db *gorm.DB, repo repository.Repository, logger *zap.Logger, c2 *Config) (*Server, error) {
	processor := bot.NewProcessor(repo, hub2, logger)
	onlineCounter := counter.NewOnlineCounter(hub2)
	unreadMessageCounter, err := counter.NewUnreadMessageCounter(db, hub2)
	if err != nil {
		return nil, err
	}
	messageCounter, err := counter.NewMessageCounter(db, hub2)
	if err != nil {
		return nil, err
	}
	channelCounter, err := counter.NewChannelCounter(db, hub2)
	if err != nil {
		return nil, err
	}
	firebaseCredentialsFilePathString := provideFirebaseCredentialsFilePathString(c2)
	client, err := newFCMClientIfAvailable(repo, logger, unreadMessageCounter, firebaseCredentialsFilePathString)
	if err != nil {
		return nil, err
	}
	manager := viewer.NewManager(hub2)
	heartbeatManager := heartbeat.NewManager(manager)
	config := provideImageProcessorConfig(c2)
	imagingProcessor := imaging.NewProcessor(config)
	streamer := sse.NewStreamer(hub2)
	webrtcv3Manager := webrtcv3.NewManager(hub2)
	wsStreamer := ws.NewStreamer(hub2, manager, webrtcv3Manager, logger)
	serverOriginString := provideServerOriginString(c2)
	notificationService := notification.NewService(repo, hub2, logger, client, streamer, wsStreamer, manager, serverOriginString)
	rbacRBAC, err := rbac.New(db)
	if err != nil {
		return nil, err
	}
	services := &service.Services{
		BOT:                  processor,
		OnlineCounter:        onlineCounter,
		UnreadMessageCounter: unreadMessageCounter,
		MessageCounter:       messageCounter,
		ChannelCounter:       channelCounter,
		FCM:                  client,
		HeartBeats:           heartbeatManager,
		Imaging:              imagingProcessor,
		Notification:         notificationService,
		RBAC:                 rbacRBAC,
		SSE:                  streamer,
		ViewerManager:        manager,
		WebRTCv3:             webrtcv3Manager,
		WS:                   wsStreamer,
	}
	routerConfig := provideRouterConfig(c2)
	echo := router.Setup(hub2, db, repo, services, logger, routerConfig)
	server := &Server{
		L:      logger,
		SS:     services,
		Router: echo,
	}
	return server, nil
}
