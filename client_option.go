package gax

type ClientOption interface {
	resolve(*clientSettings)
}

type clientOptions []ClientOption

func (opts clientOptions) resolve(s *clientSettings) *clientSettings {
	for _, opt := range opts {
		opt.resolve(s)
	}
	return s
}

type clientSettings struct {
	apiName    string
	apiVersion string
	endpoint   string
	scopes     []string
}

type withAPIName string

func (w withAPIName) resolve(s *clientSettings) {
	s.apiName = string(w)
}

func WithAPIName(apiName string) ClientOption {
	return withAPIName(apiName)
}

type withAPIVersion string

func (w withAPIVersion) resolve(s *clientSettings) {
	s.apiVersion = string(w)
}

func WithAPIVersion(apiVersion string) ClientOption {
	return withAPIVersion(apiVersion)
}

type withEndpoint string

func (w withEndpoint) resolve(s *clientSettings) {
	s.endpoint = string(w)
}

func WithEndpoint(endpoint string) ClientOption {
	return withEndpoint(endpoint)
}

type withScopes []string

func (w withScopes) resolve(s *clientSettings) {
	s.scopes = append(s.scopes[:0], w...)
}

func WithScopes(scopes ...string) ClientOption {
	return withScopes(scopes)
}
