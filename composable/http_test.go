package composable

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHttpGetDescription_Url_Https(t *testing.T) {
	httpGet := DescribeHttpGet(Https, 3457, "/health")
	assert.Equal(t, "https://localhost:3457/health", httpGet.Url())
}

func TestHttpGetDescription_Url_Http(t *testing.T) {
	httpGet := DescribeHttpGet(Http, 3457, "/health")
	assert.Equal(t, "http://localhost:3457/health", httpGet.Url())
}

func TestHttpGetDescription_Url_EmptyPath(t *testing.T) {
	httpGet := DescribeHttpGet(Https, 3457, "")
	assert.Equal(t, "https://localhost:3457", httpGet.Url())
}

func TestHttpGetDescription_Url_PrependsPathSlash(t *testing.T) {
	httpGet := DescribeHttpGet(Https, 3457, "health")
	assert.Equal(t, "https://localhost:3457/health", httpGet.Url())
}
