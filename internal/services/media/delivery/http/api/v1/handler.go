package v1

import (
	"io"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/martketplace-vkr/gateway/internal/common/httpx"
	mediaclient "github.com/martketplace-vkr/media/pkg/api/grpc/v1/media"
)

type Handler struct {
	mediaClient mediaclient.MediaServiceClient
	timeout     time.Duration
}

func New(mediaClient mediaclient.MediaServiceClient, timeout time.Duration) *Handler {
	return &Handler{
		mediaClient: mediaClient,
		timeout:     timeout,
	}
}

func (h *Handler) Upload(c *fiber.Ctx) error {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "file is required")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "could not open uploaded file")
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "could not read uploaded file")
	}

	contentType := fileHeader.Header.Get("Content-Type")
	if strings.TrimSpace(contentType) == "" {
		contentType = c.FormValue("content_type")
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.mediaClient.Upload(ctx, &mediaclient.UploadRequest{
		Filename:    fileHeader.Filename,
		ContentType: contentType,
		Content:     content,
		Directory:   c.FormValue("directory"),
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}
