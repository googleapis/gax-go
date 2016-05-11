package gax

type ClientOption interface {
	Resolve(*ClientSettings)
}

type clientOptions []ClientOption

func (opts clientOptions) Resolve(s *ClientSettings) *ClientSettings {
	for _, opt := range opts {
		opt.Resolve(s)
	}
	return s
}

type ClientSettings struct {
	APIName    string
	APIVersion string
	Endpoint   string
	Scopes     []string
}

func (w ClientSettings) Resolve(s *ClientSettings) {
	s.APIName = w.APIName
	s.APIVersion = w.APIVersion
	s.Endpoint = w.Endpoint
	s.Scopes = append([]string{}, w.Scopes...)
}

type withAPIName string

func (w withAPIName) Resolve(s *ClientSettings) {
	s.APIName = string(w)
}

func WithAPIName(apiName string) ClientOption {
	return withAPIName(apiName)
}

type withAPIVersion string

func (w withAPIVersion) Resolve(s *ClientSettings) {
	s.APIVersion = string(w)
}

func WithAPIVersion(apiVersion string) ClientOption {
	return withAPIVersion(apiVersion)
}

type withEndpoint string

func (w withEndpoint) Resolve(s *ClientSettings) {
	s.Endpoint = string(w)
}

func WithEndpoint(endpoint string) ClientOption {
	return withEndpoint(endpoint)
}

type withScopes []string

func (w withScopes) Resolve(s *ClientSettings) {
	s.Scopes = append(s.Scopes[:0], w...)
}

func WithScopes(scopes ...string) ClientOption {
	return withScopes(scopes)
}
