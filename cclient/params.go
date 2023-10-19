package cclient

import "net/url"

//////////////////////////////////////////////////

type URLParams map[string][]string

func (u URLParams) Get(key string) (value string) {
	vs := u[key]
	if len(vs) > 0 {
		value = vs[0]
	}

	return
}

func (u URLParams) Set(key, value string) {
	u[key] = []string{value}
}

func (u URLParams) SetMulti(key string, values []string) {
	u[key] = values
}

func (u URLParams) Encode() string {
	return url.Values(u).Encode()
}
