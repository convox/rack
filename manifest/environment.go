package manifest

import (
	"regexp"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

var (
	regexpInterpolation = regexp.MustCompile(`\$\{([^}]*?)\}`)
)

type Environment map[string]string

func interpolateManifest(data []byte, env Environment) ([]byte, error) {
	ds, err := interpolateServices(data)
	if err != nil {
		return nil, err
	}

	return interpolate(ds, env, true), nil
}

func interpolateServices(data []byte) ([]byte, error) {
	var m yaml.MapSlice

	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	for i, mi := range m {
		if mk, ok := mi.Key.(string); ok && mk == "services" {
			if mv, ok := mi.Value.(yaml.MapSlice); ok {
				for j, sms := range mv {
					env := Environment{}
					if smsv, ok := sms.Value.(yaml.MapSlice); ok {
						for _, smsvi := range smsv {
							if ek, ok := smsvi.Key.(string); ok && ek == "environment" {
								if ev, ok := smsvi.Value.([]interface{}); ok {
									for _, evi := range ev {
										if es, ok := evi.(string); ok {
											parts := strings.SplitN(es, "=", 2)
											if len(parts) == 2 {
												env[parts[0]] = parts[1]
											}
										}
									}
								}
							}
						}
						if len(env) > 0 {
							data, err := yaml.Marshal(sms.Value)
							if err != nil {
								return nil, err
							}
							var ms yaml.MapSlice
							if err := yaml.Unmarshal(interpolate(data, env, false), &ms); err != nil {
								return nil, err
							}
							sms.Value = ms
							mv[j] = sms
						}
					}
				}

				m[i].Value = mv
			}
		}
	}

	data, err := yaml.Marshal(m)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func interpolate(data []byte, env Environment, replaceEmpty bool) []byte {
	return regexpInterpolation.ReplaceAllFunc(data, func(m []byte) []byte {
		v, ok := env[string(m)[2:len(m)-1]]
		if ok || replaceEmpty {
			return []byte(v)
		} else {
			return m
		}
	})
}
