package httpx

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var protoMarshalOptions = protojson.MarshalOptions{
	UseProtoNames:   true,
	EmitUnpopulated: false,
}

func RPCContext(c *fiber.Ctx, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.UserContext(), timeout)
}

func WriteProtoJSON(c *fiber.Ctx, msg proto.Message) error {
	body, err := protoMarshalOptions.Marshal(msg)
	if err != nil {
		return err
	}

	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

	return c.Status(fiber.StatusOK).Send(body)
}

func MapGRPCError(err error) error {
	if err == nil {
		return nil
	}

	st, ok := grpcstatus.FromError(err)
	if !ok {
		return err
	}

	return fiber.NewError(mapGRPCCode(st.Code()), st.Message())
}

func IsGRPCCode(err error, code codes.Code) bool {
	return grpcstatus.Code(err) == code
}

func mapGRPCCode(code codes.Code) int {
	switch code {
	case codes.InvalidArgument:
		return fiber.StatusBadRequest
	case codes.Unauthenticated:
		return fiber.StatusUnauthorized
	case codes.PermissionDenied:
		return fiber.StatusForbidden
	case codes.NotFound:
		return fiber.StatusNotFound
	case codes.AlreadyExists:
		return fiber.StatusConflict
	case codes.ResourceExhausted:
		return fiber.StatusTooManyRequests
	case codes.FailedPrecondition:
		return fiber.StatusPreconditionFailed
	case codes.DeadlineExceeded:
		return fiber.StatusGatewayTimeout
	case codes.Canceled:
		return fiber.StatusRequestTimeout
	case codes.Unimplemented:
		return fiber.StatusNotImplemented
	case codes.Unavailable:
		return fiber.StatusServiceUnavailable
	default:
		return fiber.StatusInternalServerError
	}
}
