package utils

import (
	"strings"
)

func AddEnvironment(env []string, keyToValue map[string]string) []string {
	if keyToValue == nil {
		return env
	}
	keysHandled := make(map[string]struct{}, len(keyToValue))

	for i, e := range env {
		kv := strings.SplitN(e, "=", 2)
		if value, ok := keyToValue[kv[0]]; ok {
			env[i] = kv[0] + "=" + value
			keysHandled[kv[0]] = struct{}{}
		}
	}
	for key, value := range keyToValue {
		if _, alreadyHandled := keysHandled[key]; !alreadyHandled {
			env = append(env, key+"="+value)
		}
	}
	return env
}
