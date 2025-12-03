package server

import (
	"context"

	"github.com/gopcua/opcua/ua"
)

func SetMethod(n *Node, fn func(context.Context) error) {
	n.call = func(ctx context.Context, args ...*ua.Variant) ([]*ua.Variant, ua.StatusCode) {
		if len(args) > 0 {
			return nil, ua.StatusBadTooManyArguments
		}

		return nil, mapError(fn(ctx))
	}
}

func SetMethod1S[T any](n *Node, fn func(context.Context, []T) error) {
	n.call = func(ctx context.Context, args ...*ua.Variant) ([]*ua.Variant, ua.StatusCode) {
		if len(args) == 0 {
			return nil, ua.StatusBadArgumentsMissing
		}

		if len(args) > 1 {
			return nil, ua.StatusBadTooManyArguments
		}

		argVal, ok := decodeInputParameterSlice[T](args[0])
		if !ok {
			return nil, ua.StatusBadTypeMismatch
		}

		return nil, mapError(fn(ctx, argVal))
	}
}

func SetMethod1[T any](n *Node, fn func(context.Context, T) error) {
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

		return nil, mapError(fn(ctx, argVal))
	}
}

func SetMethod2[T, U any](n *Node, fn func(context.Context, T, U) error) {
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

		return nil, mapError(fn(ctx, arg0Val, arg1Val))
	}
}

func SetMethod3[T, U, V any](n *Node, fn func(context.Context, T, U, V) error) {
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

		return nil, mapError(fn(ctx, arg0Val, arg1Val, arg2Val))
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

func decodeInputParameterSlice[T any](v *ua.Variant) (val []T, ok bool) {
	if val, ok = v.Value().([]T); ok {
		return
	}

	if extObj := v.ExtensionObject(); extObj != nil {
		if val, ok = extObj.Value.([]T); ok {
			return
		}
	}

	if v.ArrayLength() > 0 {
		var eos []*ua.ExtensionObject
		if eos, ok = v.Value().([]*ua.ExtensionObject); ok {
			val = make([]T, 0, v.ArrayLength())
			for idx := range eos {
				if t, ok := eos[idx].Value.(T); ok {
					val = append(val, t)
				}
			}
			ok = len(val) == int(v.ArrayLength())
		}
	}

	return
}

func mapError(err error) ua.StatusCode {
	if err == nil {
		return ua.StatusOK
	}

	if code, ok := err.(ua.StatusCode); ok {
		return code
	}

	return ua.StatusBadUnexpectedError
}
