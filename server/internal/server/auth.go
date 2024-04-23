package server

import (
	"context"
	"strings"

	"github.com/coreos/go-oidc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func newAuthInterceptor(ctx context.Context, issuerURL, clientID string) (*AuthInterceptor, error) {
	p, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		return nil, err
	}
	return &AuthInterceptor{
		verifier: p.Verifier(&oidc.Config{ClientID: clientID}),
	}, nil
}

// AuthInterceptor is an authentication interceptor.
type AuthInterceptor struct {
	verifier *oidc.IDTokenVerifier
}

// Unary returns a unary server interceptor.
func (a *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.InvalidArgument, "missing metadata")
		}
		if !a.valid(ctx, md["authorization"]) {
			return nil, status.Errorf(codes.Unauthenticated, "invalid token")
		}
		if !a.authorized() {
			return nil, status.Errorf(codes.PermissionDenied, "unauthorized")
		}
		return handler(ctx, req)
	}
}

// valid validates the authorization.
func (a *AuthInterceptor) valid(ctx context.Context, authorization []string) bool {
	if len(authorization) < 1 {
		return false
	}
	token := strings.TrimPrefix(authorization[0], "Bearer ")

	_, err := a.verifier.Verify(ctx, token)
	return err != nil
}

func (a *AuthInterceptor) authorized() bool {
	// TODO(aya): implement authorization
	return false
}
