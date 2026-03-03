package receiver

import (
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	commonv1 "go.opentelemetry.io/proto/otlp/common/v1"
)

func attributesToMap(attrs []*commonv1.KeyValue) map[string]string {
	m := make(map[string]string, len(attrs))
	for _, kv := range attrs {
		m[kv.Key] = anyValueToString(kv.Value)
	}
	return m
}

func anyValueToString(v *commonv1.AnyValue) string {
	if v == nil {
		return ""
	}
	switch val := v.Value.(type) {
	case *commonv1.AnyValue_StringValue:
		return val.StringValue
	case *commonv1.AnyValue_BoolValue:
		return strconv.FormatBool(val.BoolValue)
	case *commonv1.AnyValue_IntValue:
		return strconv.FormatInt(val.IntValue, 10)
	case *commonv1.AnyValue_DoubleValue:
		return strconv.FormatFloat(val.DoubleValue, 'f', -1, 64)
	case *commonv1.AnyValue_BytesValue:
		return hex.EncodeToString(val.BytesValue)
	case *commonv1.AnyValue_ArrayValue:
		if val.ArrayValue == nil {
			return "[]"
		}
		parts := make([]string, 0, len(val.ArrayValue.Values))
		for _, elem := range val.ArrayValue.Values {
			parts = append(parts, anyValueToString(elem))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case *commonv1.AnyValue_KvlistValue:
		if val.KvlistValue == nil {
			return "{}"
		}
		parts := make([]string, 0, len(val.KvlistValue.Values))
		for _, kv := range val.KvlistValue.Values {
			parts = append(parts, kv.Key+"="+anyValueToString(kv.Value))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	default:
		return ""
	}
}

func traceIDToHex(id []byte) string {
	return hex.EncodeToString(id)
}

func spanIDToHex(id []byte) string {
	return hex.EncodeToString(id)
}

func nanosToTime(ns uint64) time.Time {
	if ns == 0 {
		return time.Time{}
	}
	return time.Unix(int64(ns/1e9), int64(ns%1e9)).UTC()
}

func getServiceName(attrs map[string]string) string {
	if v, ok := attrs["service.name"]; ok {
		return v
	}
	return "unknown"
}
