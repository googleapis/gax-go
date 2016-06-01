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
	AppName     string
	AppVersion  string
	Endpoint    string
	Scopes      []string
	Insecure    bool
	CallOptions map[string][]CallOption
}

func (w ClientSettings) Resolve(s *ClientSettings) {
	s.AppName = w.AppName
	s.AppVersion = w.AppVersion
	s.Endpoint = w.Endpoint
	s.Insecure = w.Insecure
	WithScopes(w.Scopes...).Resolve(s)
	WithCallOptions(w.CallOptions).Resolve(s)
}

type withAppName string

func (w withAppName) Resolve(s *ClientSettings) {
	s.AppName = string(w)
}

func WithAppName(appName string) ClientOption {
	return withAppName(appName)
}

type withAppVersion string

func (w withAppVersion) Resolve(s *ClientSettings) {
	s.AppVersion = string(w)
}

func WithAppVersion(appVersion string) ClientOption {
	return withAppVersion(appVersion)
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
	s.Scopes = append([]string{}, w...)
}

func WithScopes(scopes ...string) ClientOption {
	return withScopes(scopes)
}

type withInsecure struct{}

func (w withInsecure) Resolve(s *ClientSettings) {
	s.Insecure = true
}

func WithInsecure() ClientOption { return withInsecure{} }

type withCallOptions map[string][]CallOption

func (w withCallOptions) Resolve(s *ClientSettings) {
	s.CallOptions = make(map[string][]CallOption, len(w))
	for key, value := range w {
		s.CallOptions[key] = value
	}
}

func WithCallOptions(callOptions map[string][]CallOption) ClientOption {
	return withCallOptions(callOptions)
}
