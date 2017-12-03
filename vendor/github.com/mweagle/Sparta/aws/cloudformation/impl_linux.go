// +build linux,!lambdabinary

package cloudformation

func platformUserName() string {
	return defaultUserName()
}
