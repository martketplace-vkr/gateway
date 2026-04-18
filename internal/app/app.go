package app

import (
	"context"

	authadmin "github.com/martketplace-vkr/auth/pkg/api/grpc/v1/admin"
	authclient "github.com/martketplace-vkr/auth/pkg/api/grpc/v1/client"
	balanceclient "github.com/martketplace-vkr/balance/pkg/api/grpc/v1/client"
	catalogclient "github.com/martketplace-vkr/catalog/pkg/api/grpc/v1/client"
	orderclient "github.com/martketplace-vkr/order/pkg/api/grpc/v1/client"
	userclient "github.com/martketplace-vkr/user/pkg/api/grpc/v1/client"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/martketplace-vkr/gateway/config"
	serverCmp "github.com/martketplace-vkr/gateway/internal/app/cmps/server"
	"github.com/martketplace-vkr/gateway/internal/common/middleware"
	authv1 "github.com/martketplace-vkr/gateway/internal/services/auth/delivery/http/api/v1"
	balancev1 "github.com/martketplace-vkr/gateway/internal/services/balance/delivery/http/api/v1"
	catalogv1 "github.com/martketplace-vkr/gateway/internal/services/catalog/delivery/http/api/v1"
	orderv1 "github.com/martketplace-vkr/gateway/internal/services/order/delivery/http/api/v1"
	userv1 "github.com/martketplace-vkr/gateway/internal/services/user/delivery/http/api/v1"
	"github.com/martketplace-vkr/gateway/pkg/routes"

	"github.com/martketplace-vkr/pkg/build"
	"github.com/martketplace-vkr/pkg/server/http"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Run(ctx context.Context, cfg *config.Config) error {
	server := http.New(cfg.HTTP.Serve)

	authConn, err := dialGRPC(ctx, cfg.Auth)
	if err != nil {
		return err
	}
	defer authConn.Close()

	catalogConn, err := dialGRPC(ctx, cfg.Catalog)
	if err != nil {
		return err
	}
	defer catalogConn.Close()

	orderConn, err := dialGRPC(ctx, cfg.Order)
	if err != nil {
		return err
	}
	defer orderConn.Close()

	userConn, err := dialGRPC(ctx, cfg.User)
	if err != nil {
		return err
	}
	defer userConn.Close()

	balanceConn, err := dialGRPC(ctx, cfg.Balance)
	if err != nil {
		return err
	}
	defer balanceConn.Close()

	adminAuthCli := authadmin.NewAuthAdminServiceClient(authConn)
	authCli := authclient.NewAuthClientServiceClient(authConn)
	catalogCli := catalogclient.NewCatalogClientServiceClient(catalogConn)
	orderCli := orderclient.NewOrderClientServiceClient(orderConn)
	userCli := userclient.NewUserClientServiceClient(userConn)
	balanceCli := balanceclient.NewBalanceClientServiceClient(balanceConn)

	authMiddleware := middleware.NewAuth(authCli, adminAuthCli, cfg.Auth.Timeout.Duration)

	authBinder := authv1.NewBinder(
		server,
		authv1.New(authCli, adminAuthCli, userCli, cfg.Auth.Timeout.Duration, cfg.User.Timeout.Duration),
	)
	catalogBinder := catalogv1.NewBinder(
		server,
		authMiddleware,
		catalogv1.New(catalogCli, cfg.Catalog.Timeout.Duration),
	)
	orderBinder := orderv1.NewBinder(
		server,
		authMiddleware,
		orderv1.New(orderCli, cfg.Order.Timeout.Duration),
	)
	userBinder := userv1.NewBinder(
		server,
		authMiddleware,
		userv1.New(userCli, cfg.User.Timeout.Duration),
	)
	balanceBinder := balancev1.NewBinder(
		server,
		authMiddleware,
		balancev1.New(balanceCli, cfg.Balance.Timeout.Duration),
	)

	binder := routes.NewBinder(
		authBinder,
		catalogBinder,
		orderBinder,
		userBinder,
		balanceBinder,
	)

	httpServer := http.NewWithBinder(
		cfg.HTTP,
		server,
		binder,
		prometheus.DefaultRegisterer,
	)

	httpCmp := serverCmp.New(cfg.ServerCmp, httpServer)

	cmps := build.Components{
		httpCmp,
	}

	app, err := build.NewApp(cmps)
	if err != nil {
		return err
	}

	return build.Run(ctx, app)
}

func dialGRPC(ctx context.Context, cfg config.GRPCClient) (*grpc.ClientConn, error) {
	dialCtx, cancel := context.WithTimeout(ctx, cfg.Timeout.Duration)
	defer cancel()

	return grpc.DialContext(
		dialCtx,
		cfg.Host,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
}
