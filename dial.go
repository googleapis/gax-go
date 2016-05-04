package gax

import (
	"fmt"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
)

func DialGRPC(ctx context.Context, opts ...ClientOption) (*grpc.ClientConn, error) {
	settings := &clientSettings{}
	clientOptions(opts).resolve(settings)

	tokenSource, err := google.DefaultTokenSource(ctx, settings.scopes...)
	if err != nil {
		return nil, fmt.Errorf("google.DefaultTokenSource: %v", err)
	}
	grpcOpts := []grpc.DialOption{
		grpc.WithPerRPCCredentials(oauth.TokenSource{TokenSource: tokenSource}),
		grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")),
	}
	return grpc.Dial(settings.endpoint, grpcOpts...)
}
