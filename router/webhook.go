package router

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/labstack/echo"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/bot"
	"github.com/traPtitech/traQ/external/imagemagick"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/notification"
	"github.com/traPtitech/traQ/notification/events"
	"github.com/traPtitech/traQ/utils/thumb"
	"gopkg.in/go-playground/validator.v9"
	"gopkg.in/go-playground/webhooks.v3/github"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

type webhookForResponse struct {
	WebhookID   string    `json:"webhookID"`
	BotUserID   string    `json:"botUserId"`
	DisplayName string    `json:"displayName"`
	Description string    `json:"description"`
	IconFileID  string    `json:"iconFileId"`
	ChannelID   string    `json:"channelId"`
	Valid       bool      `json:"valid"`
	CreatorID   string    `json:"creatorId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// GetWebhooks : GET /webhooks
func (h *Handlers) GetWebhooks(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID

	list := h.Bot.GetWebhooksByCreator(uuid.FromStringOrNil(userID))
	res := make([]*webhookForResponse, len(list))
	for i, v := range list {
		res[i] = formatWebhook(&v)
	}

	return c.JSON(http.StatusOK, res)
}

// PostWebhooks : POST /webhooks
func (h *Handlers) PostWebhooks(c echo.Context) error {
	user := c.Get("user").(*model.User)

	req := struct {
		Name        string `json:"name"        form:"name"`
		Description string `json:"description" form:"description"`
		ChannelID   string `json:"channelId"   form:"channelId"`
	}{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if _, err := validateChannelID(req.ChannelID, userID); err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound, "this channel is not found")
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find the specified channel")
		}
	}

	fileID := uuid.Nil
	if c.Request().MultipartForm != nil {
		if uploadedFile, err := c.FormFile("file"); err == nil {
			// ファイルサイズ制限1MB
			if uploadedFile.Size > 1024*1024 {
				return echo.NewHTTPError(http.StatusBadRequest, "too big image file")
			}

			// ファイルタイプ確認・必要があればリサイズ
			b := &bytes.Buffer{}
			src, err := uploadedFile.Open()
			if err != nil {
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
			defer src.Close()
			switch uploadedFile.Header.Get(echo.HeaderContentType) {
			case "image/png":
				img, err := png.Decode(src)
				if err != nil {
					// 不正なpngである
					return echo.NewHTTPError(http.StatusBadRequest, "bad png file")
				}
				if img.Bounds().Size().X > iconMaxWidth || img.Bounds().Size().Y > iconMaxHeight {
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) //10秒以内に終わらないファイルは無効
					defer cancel()
					img, err = thumb.Resize(ctx, img, iconMaxWidth, iconMaxHeight)
					if err != nil {
						switch err {
						case context.DeadlineExceeded:
							// リサイズタイムアウト
							return echo.NewHTTPError(http.StatusBadRequest, "bad png file (resize timeout)")
						default:
							// 予期しないエラー
							c.Logger().Error(err)
							return echo.NewHTTPError(http.StatusInternalServerError)
						}
					}
				}

				// bytesに戻す
				if b, err = thumb.EncodeToPNG(img); err != nil {
					// 予期しないエラー
					c.Logger().Error(err)
					return echo.NewHTTPError(http.StatusInternalServerError)
				}

			case "image/jpeg":
				img, err := jpeg.Decode(src)
				if err != nil {
					// 不正なjpgである
					return echo.NewHTTPError(http.StatusBadRequest, "bad jpg file")
				}
				if img.Bounds().Size().X > iconMaxWidth || img.Bounds().Size().Y > iconMaxHeight {
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) //10秒以内に終わらないファイルは無効
					defer cancel()
					img, err = thumb.Resize(ctx, img, iconMaxWidth, iconMaxHeight)
					if err != nil {
						switch err {
						case context.DeadlineExceeded:
							// リサイズタイムアウト
							return echo.NewHTTPError(http.StatusBadRequest, "bad jpg file (resize timeout)")
						default:
							// 予期しないエラー
							c.Logger().Error(err)
							return echo.NewHTTPError(http.StatusInternalServerError)
						}
					}
				}

				// PNGに変換
				if b, err = thumb.EncodeToPNG(img); err != nil {
					// 予期しないエラー
					c.Logger().Error(err)
					return echo.NewHTTPError(http.StatusInternalServerError)
				}

			case "image/gif":
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) //10秒以内に終わらないファイルは無効
				defer cancel()
				b, err = imagemagick.ResizeAnimationGIF(ctx, src, iconMaxWidth, iconMaxHeight, false)
				if err != nil {
					switch err {
					case imagemagick.ErrUnavailable:
						// gifは一時的にサポートされていない
						return echo.NewHTTPError(http.StatusBadRequest, "gif file is temporarily unsupported")
					case imagemagick.ErrUnsupportedType:
						// 不正なgifである
						return echo.NewHTTPError(http.StatusBadRequest, "bad gif file")
					case context.DeadlineExceeded:
						// リサイズタイムアウト
						return echo.NewHTTPError(http.StatusBadRequest, "bad gif file (resize timeout)")
					default:
						// 予期しないエラー
						c.Logger().Error(err)
						return echo.NewHTTPError(http.StatusInternalServerError)
					}
				}

			case "image/svg+xml":
				// TODO svgバリデーション
				io.Copy(b, src)

			default:
				return echo.NewHTTPError(http.StatusBadRequest, "invalid image file")
			}

			// アイコン画像保存
			file := &model.File{
				Name: uploadedFile.Filename,
				Size: int64(b.Len()),
			}
			if err := file.Create(b); err != nil {
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}

			fileID = uuid.FromStringOrNil(file.ID)
		} else if err != http.ErrMissingFile {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}
	}

	w, err := h.Bot.CreateWebhook(req.Name, req.Description, uuid.FromStringOrNil(req.ChannelID), user.GetUID(), fileID)
	if err != nil {
		switch err.(type) {
		case *validator.InvalidValidationError:
			return echo.NewHTTPError(http.StatusBadRequest, err)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	go notification.Send(events.UserJoined, events.UserEvent{ID: w.ID.String()})

	return c.JSON(http.StatusCreated, formatWebhook(&w))
}

// GetWebhook : GET /webhooks/:webhookID
func (h *Handlers) GetWebhook(c echo.Context) error {
	webhookID := c.Param("webhookID")
	userID := c.Get("user").(*model.User).ID

	w, ok := h.Bot.GetWebhook(uuid.FromStringOrNil(webhookID))
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound)
	}
	if w.CreatorID != uuid.FromStringOrNil(userID) {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	return c.JSON(http.StatusOK, formatWebhook(&w))
}

// PatchWebhook : PATCH /webhooks/:webhookID
func (h *Handlers) PatchWebhook(c echo.Context) error {
	webhookID := c.Param("webhookID")
	user := c.Get("user").(*model.User)

	w, ok := h.Bot.GetWebhook(uuid.FromStringOrNil(webhookID))
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound)
	}
	if w.CreatorID != user.GetUID() {
		return echo.NewHTTPError(http.StatusForbidden)
	}
	if !w.IsValid {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	req := struct {
		Name        string `json:"name"        form:"name"`
		Description string `json:"description" form:"description"`
		ChannelID   string `json:"channelId"   form:"channelId"`
	}{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	fileID := uuid.Nil
	if c.Request().MultipartForm != nil {
		if uploadedFile, err := c.FormFile("file"); err == nil {
			// ファイルサイズ制限1MB
			if uploadedFile.Size > 1024*1024 {
				return echo.NewHTTPError(http.StatusBadRequest, "too big image file")
			}

			// ファイルタイプ確認・必要があればリサイズ
			b := &bytes.Buffer{}
			src, err := uploadedFile.Open()
			if err != nil {
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
			defer src.Close()
			switch uploadedFile.Header.Get(echo.HeaderContentType) {
			case "image/png":
				img, err := png.Decode(src)
				if err != nil {
					// 不正なpngである
					return echo.NewHTTPError(http.StatusBadRequest, "bad png file")
				}
				if img.Bounds().Size().X > iconMaxWidth || img.Bounds().Size().Y > iconMaxHeight {
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) //10秒以内に終わらないファイルは無効
					defer cancel()
					img, err = thumb.Resize(ctx, img, iconMaxWidth, iconMaxHeight)
					if err != nil {
						switch err {
						case context.DeadlineExceeded:
							// リサイズタイムアウト
							return echo.NewHTTPError(http.StatusBadRequest, "bad png file (resize timeout)")
						default:
							// 予期しないエラー
							c.Logger().Error(err)
							return echo.NewHTTPError(http.StatusInternalServerError)
						}
					}
				}

				// bytesに戻す
				if b, err = thumb.EncodeToPNG(img); err != nil {
					// 予期しないエラー
					c.Logger().Error(err)
					return echo.NewHTTPError(http.StatusInternalServerError)
				}

			case "image/jpeg":
				img, err := jpeg.Decode(src)
				if err != nil {
					// 不正なjpgである
					return echo.NewHTTPError(http.StatusBadRequest, "bad jpg file")
				}
				if img.Bounds().Size().X > iconMaxWidth || img.Bounds().Size().Y > iconMaxHeight {
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) //10秒以内に終わらないファイルは無効
					defer cancel()
					img, err = thumb.Resize(ctx, img, iconMaxWidth, iconMaxHeight)
					if err != nil {
						switch err {
						case context.DeadlineExceeded:
							// リサイズタイムアウト
							return echo.NewHTTPError(http.StatusBadRequest, "bad jpg file (resize timeout)")
						default:
							// 予期しないエラー
							c.Logger().Error(err)
							return echo.NewHTTPError(http.StatusInternalServerError)
						}
					}
				}

				// PNGに変換
				if b, err = thumb.EncodeToPNG(img); err != nil {
					// 予期しないエラー
					c.Logger().Error(err)
					return echo.NewHTTPError(http.StatusInternalServerError)
				}

			case "image/gif":
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) //10秒以内に終わらないファイルは無効
				defer cancel()
				b, err = imagemagick.ResizeAnimationGIF(ctx, src, iconMaxWidth, iconMaxHeight, false)
				if err != nil {
					switch err {
					case imagemagick.ErrUnavailable:
						// gifは一時的にサポートされていない
						return echo.NewHTTPError(http.StatusBadRequest, "gif file is temporarily unsupported")
					case imagemagick.ErrUnsupportedType:
						// 不正なgifである
						return echo.NewHTTPError(http.StatusBadRequest, "bad gif file")
					case context.DeadlineExceeded:
						// リサイズタイムアウト
						return echo.NewHTTPError(http.StatusBadRequest, "bad gif file (resize timeout)")
					default:
						// 予期しないエラー
						c.Logger().Error(err)
						return echo.NewHTTPError(http.StatusInternalServerError)
					}
				}

			case "image/svg+xml":
				// TODO svgバリデーション
				io.Copy(b, src)

			default:
				return echo.NewHTTPError(http.StatusBadRequest, "invalid image file")
			}

			// アイコン画像保存
			file := &model.File{
				Name: uploadedFile.Filename,
				Size: int64(b.Len()),
			}
			if err := file.Create(b); err != nil {
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}

			fileID = uuid.FromStringOrNil(file.ID)
		} else if err != http.ErrMissingFile {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}
	}

	if len(req.Name) > 0 {
		w.Name = req.Name
		if err := h.Bot.UpdateWebhook(&w); err != nil {
			switch err.(type) {
			case *validator.InvalidValidationError:
				return echo.NewHTTPError(http.StatusBadRequest, err)
			default:
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
		}

		go notification.Send(events.UserUpdated, events.UserEvent{ID: w.BotUserID.String()})
	}

	if len(req.Description) > 0 {
		w.Description = req.Description
		if err := h.Bot.UpdateWebhook(&w); err != nil {
			switch err.(type) {
			case *validator.InvalidValidationError:
				return echo.NewHTTPError(http.StatusBadRequest, err)
			default:
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
		}
	}

	if fileID != uuid.Nil {
		w.IconFileID = fileID
		if err := h.Bot.UpdateWebhook(&w); err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}

		go notification.Send(events.UserIconUpdated, events.UserEvent{ID: w.BotUserID.String()})
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteWebhook : DELETE /webhooks/:webhookID
func (h *Handlers) DeleteWebhook(c echo.Context) error {
	webhookID := c.Param("webhookID")
	user := c.Get("user").(*model.User)

	w, ok := h.Bot.GetWebhook(uuid.FromStringOrNil(webhookID))
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound)
	}
	if w.CreatorID != user.GetUID() {
		return echo.NewHTTPError(http.StatusForbidden)
	}
	if !w.IsValid {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	w.IsValid = false
	if err := h.Bot.UpdateWebhook(&w); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// PostWebhook : POST /webhooks/:webhookID
func (h *Handlers) PostWebhook(c echo.Context) error {
	webhookID := c.Param("webhookID")

	w, ok := h.Bot.GetWebhook(uuid.FromStringOrNil(webhookID))
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound)
	}
	if !w.IsValid {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	message := &model.Message{
		UserID:    w.BotUserID.String(),
		ChannelID: w.ChannelID.String(),
	}
	switch c.Request().Header.Get(echo.HeaderContentType) {
	case echo.MIMETextPlain, echo.MIMETextPlainCharsetUTF8:
		if b, err := ioutil.ReadAll(c.Request().Body); err == nil {
			message.Text = string(b)
		}
		if len(message.Text) == 0 {
			return echo.NewHTTPError(http.StatusBadRequest)
		}

	case echo.MIMEApplicationJSON, echo.MIMEApplicationForm, echo.MIMEApplicationJSONCharsetUTF8:
		req := struct {
			Text      string `json:"text"      form:"text"`
			ChannelID string `json:"channelId" form:"channelId"`
		}{}
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}
		if len(req.Text) == 0 {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		if len(req.ChannelID) == 36 {
			message.ChannelID = req.ChannelID
		}
		message.Text = req.Text

	default:
		return echo.NewHTTPError(http.StatusUnsupportedMediaType)
	}

	if err := message.Create(); err != nil {
		if errSQL, ok := err.(*mysql.MySQLError); ok {
			if errSQL.Number == 1452 { //外部キー制約
				return echo.NewHTTPError(http.StatusBadRequest, "invalid channelId")
			}
		}

		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go notification.Send(events.MessageCreated, events.MessageEvent{Message: *message})
	return c.NoContent(http.StatusNoContent)
}

// PostWebhookByGithub : POST /webhooks/:webhookID/github
func (h *Handlers) PostWebhookByGithub(c echo.Context) error {
	webhookID := c.Param("webhookID")

	w, ok := h.Bot.GetWebhook(uuid.FromStringOrNil(webhookID))
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound)
	}
	if !w.IsValid {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	switch c.Request().Header.Get(echo.HeaderContentType) {
	case echo.MIMEApplicationJSON, echo.MIMEApplicationJSONCharsetUTF8:
		break
	default:
		return echo.NewHTTPError(http.StatusUnsupportedMediaType)
	}

	event := c.Request().Header.Get("X-GitHub-Event")
	if len(event) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "missing X-GitHub-Event header")
	}

	githubEvent := github.Event(event)

	//MEMO 現在はサーバー側で簡単に整形してるけど、将来的にクライアント側に表示デザイン込みで任せたいよね
	message := &model.Message{
		UserID:    w.BotUserID.String(),
		ChannelID: w.ChannelID.String(),
	}

	switch githubEvent {
	case github.IssuesEvent: // Any time an Issue is assigned, unassigned, labeled, unlabeled, opened, edited, milestoned, demilestoned, closed, or reopened.
		var i github.IssuesPayload
		if err := json.NewDecoder(c.Request().Body).Decode(&i); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}

		switch i.Action {
		case "opened":
			message.Text = fmt.Sprintf("## Issue Opened\n[%s](%s) - [%s](%s)", i.Repository.FullName, i.Repository.HTMLURL, i.Issue.Title, i.Issue.HTMLURL)
		case "closed":
			message.Text = fmt.Sprintf("## Issue Closed\n[%s](%s) - [%s](%s)", i.Repository.FullName, i.Repository.HTMLURL, i.Issue.Title, i.Issue.HTMLURL)
		case "reopened":
			message.Text = fmt.Sprintf("## Issue Reopened\n[%s](%s) - [%s](%s)", i.Repository.FullName, i.Repository.HTMLURL, i.Issue.Title, i.Issue.HTMLURL)
		case "assigned", "unassigned", "labeled", "unlabeled", "edited", "milestoned", "demilestoned":
			// Unsupported
		}

	case github.PullRequestEvent: // Any time a pull request is assigned, unassigned, labeled, unlabeled, opened, edited, closed, reopened, or synchronized (updated due to a new push in the branch that the pull request is tracking). Also any time a pull request review is requested, or a review request is removed.
		var p github.PullRequestPayload
		if err := json.NewDecoder(c.Request().Body).Decode(&p); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}

		switch p.Action {
		case "opened":
			message.Text = fmt.Sprintf("## PullRequest Opened\n[%s](%s) - [%s](%s)", p.Repository.FullName, p.Repository.HTMLURL, p.PullRequest.Title, p.PullRequest.HTMLURL)
		case "closed":
			if p.PullRequest.Merged {
				message.Text = fmt.Sprintf("## PullRequest Merged\n[%s](%s) - [%s](%s)", p.Repository.FullName, p.Repository.HTMLURL, p.PullRequest.Title, p.PullRequest.HTMLURL)
			} else {
				message.Text = fmt.Sprintf("## PullRequest Closed\n[%s](%s) - [%s](%s)", p.Repository.FullName, p.Repository.HTMLURL, p.PullRequest.Title, p.PullRequest.HTMLURL)
			}
		case "reopened":
			message.Text = fmt.Sprintf("## PullRequest Reopened\n[%s](%s) - [%s](%s)", p.Repository.FullName, p.Repository.HTMLURL, p.PullRequest.Title, p.PullRequest.HTMLURL)
		case "assigned", "unassigned", "labeled", "unlabeled", "edited", "review_requested", "review_request_removed":
			// Unsupported
		}

	case github.PushEvent: // Any Git push to a Repository, including editing tags or branches. Commits via API actions that update references are also counted. This is the default event.
		var p github.PushPayload
		if err := json.NewDecoder(c.Request().Body).Decode(&p); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}

		message.Text = fmt.Sprintf("## %d Commit(s) Pushed by %s\n[%s](%s) , refs: `%s`, [compare](%s)\n", len(p.Commits), p.Pusher.Name, p.Repository.FullName, p.Repository.HTMLURL, p.Ref, p.Compare)

		for _, v := range p.Commits {
			message.Text += fmt.Sprintf("+ [`%7s`](%s) - %s \n", v.ID, v.URL, v.Message)
		}

	default:
		// Currently Unsupported:
		// marketplace_purchase, fork, gollum, installation, installation_repositories, label, ping, member, membership,
		// organization, org_block, page_build, public, repository, status, team, team_add, watch, create, delete, deployment,
		// deployment_status, project_column, milestone, project_card, project, commit_comment, release, issue_comment,
		// pull_request_review, pull_request_review_comment
		// 上ので必要な場合は実装してプルリクを飛ばしてください
	}

	if len(message.Text) > 0 {
		if err := message.Create(); err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
		go notification.Send(events.MessageCreated, events.MessageEvent{Message: *message})
	}

	return c.NoContent(http.StatusNoContent)
}

func formatWebhook(w *bot.Webhook) *webhookForResponse {
	return &webhookForResponse{
		WebhookID:   w.ID.String(),
		BotUserID:   w.BotUserID.String(),
		DisplayName: w.Name,
		Description: w.Description,
		IconFileID:  w.IconFileID.String(),
		ChannelID:   w.ChannelID.String(),
		Valid:       w.IsValid,
		CreatorID:   w.CreatorID.String(),
		CreatedAt:   w.CreatedAt,
		UpdatedAt:   w.UpdatedAt,
	}
}
