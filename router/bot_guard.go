package router

import (
	"github.com/gofrs/uuid"
	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
	"go.uber.org/zap"
	"net/http"
)

type botGuardFunc func(h *Handlers, bot *model.Bot, c echo.Context) (bool, error)

// BotGuard Botのリクエストを制限するミドルウェア. PrivilegedなBOTは制限されない
func (h *Handlers) BotGuard(f botGuardFunc) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user := getRequestUser(c)
			if !user.Bot {
				return next(c)
			}

			b, err := h.Repo.GetBotByBotUserID(user.ID)
			if err != nil {
				h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
				return echo.NewHTTPError(http.StatusInternalServerError)
			}

			if b.Privileged {
				return next(c)
			}

			ok, err := f(h, b, c)
			if err != nil {
				h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
			if !ok {
				return echo.NewHTTPError(http.StatusForbidden)
			}

			return next(c)
		}
	}
}

// blockAlways 常にBOTのリクエストを拒否
func blockAlways(h *Handlers, bot *model.Bot, c echo.Context) (bool, error) {
	return true, nil
}

// blockByChannelIdQuery BOTが参加しているチャンネル以外へのリクエストを拒否
func blockByChannelIdQuery(h *Handlers, bot *model.Bot, c echo.Context) (bool, error) {
	return blockByChannelId(h, bot, c, getRequestParamAsUUID(c, paramChannelID))
}

// blockByMessageChannel BOTが参加しているチャンネル以外のメッセージへのリクエストを拒否
func blockByMessageChannel(h *Handlers, bot *model.Bot, c echo.Context) (bool, error) {
	return blockByChannelId(h, bot, c, getMessageFromContext(c).ChannelID)
}

func blockByChannelId(h *Handlers, bot *model.Bot, c echo.Context, channelId uuid.UUID) (bool, error) {
	ids, err := h.Repo.GetParticipatingChannelIDsByBot(bot.ID)
	if err != nil {
		return false, err
	}

	for _, v := range ids {
		if v == channelId {
			return true, nil
		}
	}
	return false, nil
}
