package composable

import "fmt"

// HttpProtocol constants for HTTP and HTTPS protocol schemes.
type HttpProtocol string

const (
	Http  HttpProtocol = "http"
	Https HttpProtocol = "https"
)

// HttpGetDescription describes a http get request with Protocol, Port and Path.
type HttpGetDescription struct {
	Protocol HttpProtocol
	Port     uint16
	Path     string
}

// DescribeHttpGet creates an HttpGetDescription.
func DescribeHttpGet(protocol HttpProtocol, port uint16, path string) *HttpGetDescription {
	if len(path) != 0 && path[0] != '/' {
		path = "/" + path
	}
	return &HttpGetDescription{
		protocol,
		port,
		path,
	}
}

// Url creates an RFC3986 URI string from HttpGetDescription fields.
func (hgd *HttpGetDescription) Url() string {
	return fmt.Sprintf("%s://localhost:%d%s", hgd.Protocol, hgd.Port, hgd.Path)
}
