package app

import (
	"context"

	analyticsvendor "github.com/martketplace-vkr/analytics/pkg/api/grpc/v1/vendor"
	authadmin "github.com/martketplace-vkr/auth/pkg/api/grpc/v1/admin"
	authclient "github.com/martketplace-vkr/auth/pkg/api/grpc/v1/client"
	authvendor "github.com/martketplace-vkr/auth/pkg/api/grpc/v1/vendor"
	balanceadmin "github.com/martketplace-vkr/balance/pkg/api/grpc/v1/admin"
	balanceclient "github.com/martketplace-vkr/balance/pkg/api/grpc/v1/client"
	cartclient "github.com/martketplace-vkr/cart/pkg/api/grpc/v1/client"
	catalogadmin "github.com/martketplace-vkr/catalog/pkg/api/grpc/v1/admin"
	catalogclient "github.com/martketplace-vkr/catalog/pkg/api/grpc/v1/client"
	catalogvendor "github.com/martketplace-vkr/catalog/pkg/api/grpc/v1/vendor"
	mediaclient "github.com/martketplace-vkr/media/pkg/api/grpc/v1/media"
	orderclient "github.com/martketplace-vkr/order/pkg/api/grpc/v1/client"
	ordervendor "github.com/martketplace-vkr/order/pkg/api/grpc/v1/vendor"
	reviewadmin "github.com/martketplace-vkr/review/pkg/api/grpc/v1/admin"
	reviewclient "github.com/martketplace-vkr/review/pkg/api/grpc/v1/client"
	reviewvendor "github.com/martketplace-vkr/review/pkg/api/grpc/v1/vendor"
	userclient "github.com/martketplace-vkr/user/pkg/api/grpc/v1/client"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/martketplace-vkr/gateway/config"
	serverCmp "github.com/martketplace-vkr/gateway/internal/app/cmps/server"
	"github.com/martketplace-vkr/gateway/internal/common/middleware"
	analyticsv1 "github.com/martketplace-vkr/gateway/internal/services/analytics/delivery/http/api/v1"
	authv1 "github.com/martketplace-vkr/gateway/internal/services/auth/delivery/http/api/v1"
	balancev1 "github.com/martketplace-vkr/gateway/internal/services/balance/delivery/http/api/v1"
	cartv1 "github.com/martketplace-vkr/gateway/internal/services/cart/delivery/http/api/v1"
	catalogv1 "github.com/martketplace-vkr/gateway/internal/services/catalog/delivery/http/api/v1"
	mediav1 "github.com/martketplace-vkr/gateway/internal/services/media/delivery/http/api/v1"
	orderv1 "github.com/martketplace-vkr/gateway/internal/services/order/delivery/http/api/v1"
	reviewv1 "github.com/martketplace-vkr/gateway/internal/services/review/delivery/http/api/v1"
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

	cartConn, err := dialGRPC(ctx, cfg.Cart)
	if err != nil {
		return err
	}
	defer cartConn.Close()

	orderConn, err := dialGRPC(ctx, cfg.Order)
	if err != nil {
		return err
	}
	defer orderConn.Close()

	analyticsConn, err := dialGRPC(ctx, cfg.Analytics)
	if err != nil {
		return err
	}
	defer analyticsConn.Close()

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

	mediaConn, err := dialGRPC(ctx, cfg.Media)
	if err != nil {
		return err
	}
	defer mediaConn.Close()

	reviewConn, err := dialGRPC(ctx, cfg.Review)
	if err != nil {
		return err
	}
	defer reviewConn.Close()

	adminAuthCli := authadmin.NewAuthAdminServiceClient(authConn)
	authCli := authclient.NewAuthClientServiceClient(authConn)
	vendorAuthCli := authvendor.NewAuthVendorServiceClient(authConn)
	cartCli := cartclient.NewCartClientServiceClient(cartConn)
	catalogAdminCli := catalogadmin.NewCatalogAdminServiceClient(catalogConn)
	catalogCli := catalogclient.NewCatalogClientServiceClient(catalogConn)
	catalogVendorCli := catalogvendor.NewCatalogVendorServiceClient(catalogConn)
	orderCli := orderclient.NewOrderClientServiceClient(orderConn)
	orderVendorCli := ordervendor.NewOrderVendorServiceClient(orderConn)
	analyticsVendorCli := analyticsvendor.NewAnalyticsVendorServiceClient(analyticsConn)
	userCli := userclient.NewUserClientServiceClient(userConn)
	balanceCli := balanceclient.NewBalanceClientServiceClient(balanceConn)
	balanceAdminCli := balanceadmin.NewBalanceAdminServiceClient(balanceConn)
	mediaCli := mediaclient.NewMediaServiceClient(mediaConn)
	reviewClientCli := reviewclient.NewReviewClientServiceClient(reviewConn)
	reviewVendorCli := reviewvendor.NewReviewVendorServiceClient(reviewConn)
	reviewAdminCli := reviewadmin.NewReviewAdminServiceClient(reviewConn)

	authMiddleware := middleware.NewAuth(authCli, adminAuthCli, vendorAuthCli, cfg.Auth.Timeout.Duration)

	authBinder := authv1.NewBinder(
		server,
		authv1.New(authCli, adminAuthCli, vendorAuthCli, userCli, cfg.Auth.Timeout.Duration, cfg.User.Timeout.Duration),
	)
	catalogBinder := catalogv1.NewBinder(
		server,
		authMiddleware,
		catalogv1.New(catalogCli, catalogAdminCli, catalogVendorCli, cfg.Catalog.Timeout.Duration),
	)
	cartBinder := cartv1.NewBinder(
		server,
		authMiddleware,
		cartv1.New(cartCli, cfg.Cart.Timeout.Duration),
	)
	orderBinder := orderv1.NewBinder(
		server,
		authMiddleware,
		orderv1.New(orderCli, orderVendorCli, cartCli, catalogCli, cfg.Order.Timeout.Duration),
	)
	analyticsBinder := analyticsv1.NewBinder(
		server,
		authMiddleware,
		analyticsv1.New(analyticsVendorCli, cfg.Analytics.Timeout.Duration),
	)
	userBinder := userv1.NewBinder(
		server,
		authMiddleware,
		userv1.New(userCli, cfg.User.Timeout.Duration),
	)
	balanceBinder := balancev1.NewBinder(
		server,
		authMiddleware,
		balancev1.New(balanceCli, balanceAdminCli, cfg.Balance.Timeout.Duration, cfg.MockProvider.InternalURL, cfg.MockProvider.WebhookSecret),
	)
	mediaBinder := mediav1.NewBinder(
		server,
		authMiddleware,
		mediav1.New(mediaCli, cfg.Media.Timeout.Duration),
	)
	reviewBinder := reviewv1.NewBinder(
		server,
		authMiddleware,
		reviewv1.New(reviewClientCli, reviewVendorCli, reviewAdminCli, cfg.Review.Timeout.Duration),
	)

	binder := routes.NewBinder(
		authBinder,
		cartBinder,
		catalogBinder,
		orderBinder,
		analyticsBinder,
		userBinder,
		balanceBinder,
		mediaBinder,
		reviewBinder,
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
