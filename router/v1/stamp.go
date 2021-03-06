package v1

import (
	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/router/utils"
	"github.com/traPtitech/traQ/service/rbac/permission"
	"github.com/traPtitech/traQ/utils/optional"
	"net/http"
)

// GetStamps GET /stamps
func (h *Handlers) GetStamps(c echo.Context) error {
	b, _, err := h.Repo.GetStampsJSON(false)
	if err != nil {
		return herror.InternalServerError(err)
	}
	return c.JSONBlob(http.StatusOK, b)
}

// PostStamp POST /stamps
func (h *Handlers) PostStamp(c echo.Context) error {
	userID := getRequestUserID(c)

	// スタンプ画像保存
	fileID, err := utils.SaveUploadStampImage(h.Imaging, c, h.FileManager, "file")
	if err != nil {
		return err
	}

	// スタンプ作成
	s, err := h.Repo.CreateStamp(repository.CreateStampArgs{Name: c.FormValue("name"), FileID: fileID, CreatorID: userID})
	if err != nil {
		switch {
		case repository.IsArgError(err):
			return herror.BadRequest(err)
		case err == repository.ErrAlreadyExists:
			return herror.Conflict("this name has already been used")
		default:
			return herror.InternalServerError(err)
		}
	}
	return c.JSON(http.StatusCreated, s)
}

// GetStamp GET /stamps/:stampID
func (h *Handlers) GetStamp(c echo.Context) error {
	stamp := getStampFromContext(c)
	return c.JSON(http.StatusOK, stamp)
}

// PatchStamp PATCH /stamps/:stampID
func (h *Handlers) PatchStamp(c echo.Context) error {
	user := getRequestUser(c)
	stampID := getRequestParamAsUUID(c, consts.ParamStampID)
	stamp := getStampFromContext(c)

	// ユーザー確認
	if stamp.CreatorID != user.GetID() && !h.RBAC.IsGranted(user.GetRole(), permission.EditStampCreatedByOthers) {
		return herror.Forbidden("you are not permitted to edit stamp created by others")
	}

	args := repository.UpdateStampArgs{}

	// 名前変更
	name := c.FormValue("name")
	if len(name) > 0 {
		args.Name = optional.StringFrom(name)
	}

	// 画像変更
	f, _, err := c.Request().FormFile("file")
	if err == nil {
		f.Close()
		fileID, err := utils.SaveUploadStampImage(h.Imaging, c, h.FileManager, "file")
		if err != nil {
			return err
		}
		args.FileID = optional.UUIDFrom(fileID)
	} else if err != http.ErrMissingFile {
		return herror.BadRequest(err)
	}

	// 更新
	if err := h.Repo.UpdateStamp(stampID, args); err != nil {
		switch {
		case repository.IsArgError(err):
			return herror.BadRequest(err)
		case err == repository.ErrAlreadyExists:
			return herror.Conflict("this name has already been used")
		default:
			return herror.InternalServerError(err)
		}
	}
	return c.NoContent(http.StatusNoContent)
}

// DeleteStamp DELETE /stamps/:stampID
func (h *Handlers) DeleteStamp(c echo.Context) error {
	stampID := getRequestParamAsUUID(c, consts.ParamStampID)

	if err := h.Repo.DeleteStamp(stampID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetMessageStamps GET /messages/:messageID/stamps
func (h *Handlers) GetMessageStamps(c echo.Context) error {
	return c.JSON(http.StatusOK, getMessageFromContext(c).Stamps)
}

// PostMessageStampRequest POST /messages/:messageID/stamps/:stampID リクエストボディ
type PostMessageStampRequest struct {
	Count int `json:"count"`
}

func (r *PostMessageStampRequest) Validate() error {
	if r.Count == 0 {
		r.Count = 1
	}
	return vd.ValidateStruct(r,
		vd.Field(&r.Count, vd.Min(1), vd.Max(100)),
	)
}

// PostMessageStamp POST /messages/:messageID/stamps/:stampID
func (h *Handlers) PostMessageStamp(c echo.Context) error {
	userID := getRequestUserID(c)
	messageID := getRequestParamAsUUID(c, consts.ParamMessageID)
	stampID := getRequestParamAsUUID(c, consts.ParamStampID)

	var req PostMessageStampRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	// スタンプをメッセージに押す
	if _, err := h.Repo.AddStampToMessage(messageID, stampID, userID, req.Count); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteMessageStamp DELETE /messages/:messageID/stamps/:stampID
func (h *Handlers) DeleteMessageStamp(c echo.Context) error {
	userID := getRequestUserID(c)
	messageID := getRequestParamAsUUID(c, consts.ParamMessageID)
	stampID := getRequestParamAsUUID(c, consts.ParamStampID)

	// スタンプをメッセージから削除
	if err := h.Repo.RemoveStampFromMessage(messageID, stampID, userID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetMyStampHistory GET /users/me/stamp-history
func (h *Handlers) GetMyStampHistory(c echo.Context) error {
	userID := getRequestUserID(c)

	history, err := h.Repo.GetUserStampHistory(userID, 100)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusOK, history)
}
