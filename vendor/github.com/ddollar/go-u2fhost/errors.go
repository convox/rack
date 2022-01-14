package u2fhost

// A TestOfUserPresenceRequiredError indicates that the device is requesting the
// user interact with it (such as pressing a button) to fulfill the given request.
type TestOfUserPresenceRequiredError struct{}

func (e TestOfUserPresenceRequiredError) Error() string {
	return "Device is requesting test of use presence to fulfill the request."
}

// A BadKeyHandleError indicates the key handle was not created by the device,
// or was created with a different application parameter.
type BadKeyHandleError struct{}

func (e BadKeyHandleError) Error() string {
	// TODO: get the actual reason from the device response.
	return "The provided key handle is not present on the device, or was created with a different application parameter."
}
