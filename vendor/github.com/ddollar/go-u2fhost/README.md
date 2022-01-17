# u2fhost
[![GoDoc](https://godoc.org/github.com/marshallbrekka/go-u2fhost?status.svg)](http://godoc.org/github.com/marshallbrekka/go-u2fhost) [![CircleCI](https://circleci.com/gh/marshallbrekka/go-u2fhost.svg?style=shield)](https://circleci.com/gh/marshallbrekka/go-u2fhost) [![codecov](https://codecov.io/gh/marshallbrekka/go-u2fhost/branch/master/graph/badge.svg)](https://codecov.io/gh/marshallbrekka/go-u2fhost)


A library for using U2F USB devices from Go programs.

## Who is this for
This library allows clients to interface with U2F USB devices to perform user authentication.

Because U2F is supported in most major browsers (either natively or by extensions), the only place I really foresee this being used (and why I wrote it in the first place) is to add U2F support to CLI apps.

## Usage

### Registration

To register with a new device, you will need to construct a `RegistrationRequest`.
```go
request := &RegisterRequest{
	// The challenge is provided by the server
	Challenge: "randomstringprovidedbyserver",
	// "The facet should be provided by the client making the request
	Facet:	 "https://example.com",
	// "The AppId may be provided by the server or the client client making the request.
	AppId:	 "https://example.com",
}
```

Next, get a list of devices that you can attempt to register with.

```go
allDevices := Devices()
// Filter only the devices that can be opened.
openDevices := []Device{}
for i, device := range devices {
	err := device.Open()
	if err == nil {
		openDevices = append(openDevices, devices[i])
		defer func(i int) {
			devices[i].Close()
		}(i)
	}
}
```

Once you have a slice of open devices, repeatedly call the `Register` function until the user activates a device, or you time out waiting for the user.

```go
// Prompt the user to perform the registration request.
fmt.Println("\nTouch the U2F device you wish to register...")
var response RegisterResponse
var err error
timeout := time.After(time.Second * 25)
interval := time.NewTicker(time.Millisecond * 250)
defer interval.Stop()
for {
    select {
    case <-timeout:
		fmt.Println("Failed to get registration response after 25 seconds")
		break
    case <-interval:
		for _, device := range openDevices {
			response, err := device.Register(req)
			if err != nil {
				if _, ok := err.(TestOfUserPresenceRequiredError); ok {
					continue
				} else {
					// you should handle errors more gracefully than this
					panic(err)
				}
			} else {
				return response
			}
		}
    }
}
```

Once you have a registration response, send the results back to your server in the form it expects.

### Authentication

To authenticate with a device, you will need to construct a `AuthenticateRequest`.

```go
request := &AuthenticateRequest{
	// The challenge is provided by the server
	Challenge: "randomstringprovidedbytheserver",
	 // "The facet should be provided by the client making the request
	Facet:	 "https://example.com",
	// "The AppId may be provided by the server or the client client making the request.
	AppId:	 "https://example.com",
	// The KeyHandle is provided by the server
	KeyHandle: "keyhandleprovidedbytheserver",
}
```

Next, get a list of devices that you can attempt to authenticate with.

```go
allDevices := Devices()
// Filter only the devices that can be opened.
openDevices := []Device{}
for i, device := range devices {
	err := device.Open()
	if err == nil {
		openDevices = append(openDevices, devices[i])
		defer func(i int) {
				devices[i].Close()
		}(i)
	}
}
```

Once you have a slice of open devices, repeatedly call the `Authenticate` function until the user activates a device, or you time out waiting for the user.

```go
prompted := false
timeout := time.After(time.Second * 25)
interval := time.NewTicker(time.Millisecond * 250)
defer interval.Stop()
for {
    select {
	case <-timeout:
		fmt.Println("Failed to get authentication response after 25 seconds")
		break
	case <-interval.C:
		for _, device := range openDevices {
			response, err := device.Authenticate(req)
			if err == nil {
				return response
				log.Debugf("Got error from device, skipping: %s", err)
			} else if _, ok := err.(TestOfUserPresenceRequiredError); ok && !prompted {
				fmt.Println("\nTouch the flashing U2F device to authenticate...\n")
				prompted = true
			} else {
				fmt.Printf("Got status response %#x\n", err)
			}
		}
    }
}
```
## Example
The `cmd` directory contains a sample CLI program that allows you to run the `register` and `authenticate` operations, providing all of the inputs that would normally be provided by the server via command line flags.

## Known issues/FAQ

### What platforms has this been tested on?
At the moment only Mac OS, however nothing in the go codebase is platform specific, and the hid library supports Mac, Windows, and Linux, so in theory it should work on all platforms.

### Linux
If you are using a linux device you need to add [these udev rules](https://github.com/Yubico/libu2f-host/blob/master/70-u2f.rules).

### The interface seems too low level, why isn't it easier to use?
Mostly because I wasn't sure what a good high level API would look like, and opted to provide a more general purpose low level API.

That said, in the future I may add a high level API similar to the Javascript U2F API.
