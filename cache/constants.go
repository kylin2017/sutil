package cache

const (
	SpanLogKeyKey    = "key"
	SpanLogCacheType = "cache"
	SpanLogOp        = "op"
)

type CacheType int

const (
	CacheTypeRedis CacheType = iota
)

func (t CacheType) String() string {
	switch t {
	case CacheTypeRedis:
		return "redis"
	default:
		return ""
	}
}

type ConfigerType int

const (
	ConfigerTypeSimple ConfigerType = iota
	ConfigerTypeEtcd
	ConfigerTypeApollo
)

func (c ConfigerType) String() string {
	switch c {
	case ConfigerTypeSimple:
		return "simple"
	case ConfigerTypeEtcd:
		return "etcd"
	case ConfigerTypeApollo:
		return "apollo"
	default:
		return "unkown"
	}
}

const DefaultRouteGroup = "default"

const (
	WrapperTypeCache    = "c"
	WrapperTypeRedisExt = "e"
)
