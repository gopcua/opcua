package context

import (
	"context"
)

type srvCtxKeyType struct{}

var srvCtxKey = srvCtxKeyType{}

type srvctx struct {
	methodID         string
	methodName       string
	methodObjectID   string
	methodObjectName string

	serviceSet  string
	serviceName string
}

func load(ctx context.Context) *srvctx {
	sc, ok := ctx.Value(srvCtxKey).(*srvctx)

	if !ok {
		return &srvctx{}
	}

	return sc
}

func store(ctx context.Context, sc *srvctx) context.Context {
	return context.WithValue(ctx, srvCtxKey, sc)
}

func WithMethodCall(ctx context.Context, objectID, objectName, methodID, methodName string) context.Context {
	sc := load(ctx)

	sc.methodID = methodID
	sc.methodName = methodName
	sc.methodObjectID = objectID
	sc.methodObjectName = objectName

	return store(ctx, sc)
}

func MethodID(ctx context.Context) string {
	return load(ctx).methodID
}

func MethodName(ctx context.Context) string {
	return load(ctx).methodName
}

func MethodObjectID(ctx context.Context) string {
	return load(ctx).methodObjectID
}

func MethodObjectName(ctx context.Context) string {
	return load(ctx).methodObjectName
}

func WithServiceSetAndName(ctx context.Context, set, name string) context.Context {
	sc := load(ctx)

	sc.serviceName = name
	sc.serviceSet = set

	return store(ctx, sc)
}

func ServiceSet(ctx context.Context) string {
	return load(ctx).serviceSet
}

func ServiceName(ctx context.Context) string {
	return load(ctx).serviceName
}
