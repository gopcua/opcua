package server

import (
	"context"

	"github.com/gopcua/opcua/ua"
)

func SetMethod(n *Node, fn func(context.Context) ua.StatusCode) {
	n.call = func(ctx context.Context, args ...*ua.Variant) ([]*ua.Variant, ua.StatusCode) {
		if len(args) > 0 {
			return nil, ua.StatusBadTooManyArguments
		}

		return nil, fn(ctx)
	}
}

func SetMethod1[T any](n *Node, fn func(context.Context, T) ua.StatusCode) {
	n.call = func(ctx context.Context, args ...*ua.Variant) ([]*ua.Variant, ua.StatusCode) {
		if len(args) == 0 {
			return nil, ua.StatusBadArgumentsMissing
		}

		if len(args) > 1 {
			return nil, ua.StatusBadTooManyArguments
		}

		argVal, ok := decodeInputParameter[T](args[0])
		if !ok {
			return nil, ua.StatusBadTypeMismatch
		}

		return nil, fn(ctx, argVal)
	}
}

func SetMethod2[T, U any](n *Node, fn func(context.Context, T, U) ua.StatusCode) {
	n.call = func(ctx context.Context, args ...*ua.Variant) ([]*ua.Variant, ua.StatusCode) {
		if len(args) < 2 {
			return nil, ua.StatusBadArgumentsMissing
		}

		if len(args) > 2 {
			return nil, ua.StatusBadTooManyArguments
		}

		arg0Val, ok0 := decodeInputParameter[T](args[0])
		arg1Val, ok1 := decodeInputParameter[U](args[1])
		if !ok0 || !ok1 {
			return nil, ua.StatusBadTypeMismatch
		}

		return nil, fn(ctx, arg0Val, arg1Val)
	}
}

func SetMethod3[T, U, V any](n *Node, fn func(context.Context, T, U, V) ua.StatusCode) {
	n.call = func(ctx context.Context, args ...*ua.Variant) ([]*ua.Variant, ua.StatusCode) {
		if len(args) < 3 {
			return nil, ua.StatusBadArgumentsMissing
		}

		if len(args) > 3 {
			return nil, ua.StatusBadTooManyArguments
		}

		arg0Val, ok0 := decodeInputParameter[T](args[0])
		arg1Val, ok1 := decodeInputParameter[U](args[1])
		arg2Val, ok2 := decodeInputParameter[V](args[1])
		if !ok0 || !ok1 || !ok2 {
			return nil, ua.StatusBadTypeMismatch
		}

		return nil, fn(ctx, arg0Val, arg1Val, arg2Val)
	}
}

func decodeInputParameter[T any](v *ua.Variant) (val T, ok bool) {
	if val, ok = v.Value().(T); ok {
		return
	}

	if extObj := v.ExtensionObject(); extObj != nil {
		if val, ok = extObj.Value.(T); ok {
			return
		}

		var ptr *T
		if ptr, ok = extObj.Value.(*T); ok {
			if ptr != nil {
				val = *ptr
			}
		}
	}

	return
}
