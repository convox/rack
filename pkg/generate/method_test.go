package generate_test

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/convox/rack/pkg/generate"
	"github.com/stretchr/testify/require"
)

func TestInts(t *testing.T) {
	testData := []struct {
		m            *generate.Method
		expectedArgs []generate.Arg
	}{
		{
			m: &generate.Method{
				Args: []generate.Arg{
					{
						Name: "arg1",
						Type: reflect.TypeOf(int(10)),
					},
					{
						Name: "arg2",
						Type: reflect.TypeOf(""),
					},
					{
						Name: "arg3",
						Type: reflect.TypeOf(int(11)),
					},
				},
			},
			expectedArgs: []generate.Arg{
				{
					Name: "arg1",
					Type: reflect.TypeOf(int(10)),
				},
				{
					Name: "arg3",
					Type: reflect.TypeOf(int(11)),
				},
			},
		},
		{
			m: &generate.Method{
				Args: []generate.Arg{
					{
						Name: "arg2",
						Type: reflect.TypeOf(""),
					},
				},
			},
			expectedArgs: []generate.Arg{},
		},
	}

	for _, td := range testData {
		require.Equal(t, td.expectedArgs, td.m.Ints())
	}
}

func TestOption(t *testing.T) {
	testData := []struct {
		m           *generate.Method
		expectedArg *generate.Arg
	}{
		{
			m: &generate.Method{
				Args: []generate.Arg{
					{
						Name: "arg1",
						Type: reflect.TypeOf(int(10)),
					},
					{
						Name: "arg2",
						Type: reflect.TypeOf(struct{}{}),
					},
					{
						Name: "arg3",
						Type: reflect.TypeOf(int(11)),
					},
				},
			},
			expectedArg: &generate.Arg{
				Name: "arg2",
				Type: reflect.TypeOf(struct{}{}),
			},
		},
		{
			m: &generate.Method{
				Args: []generate.Arg{
					{
						Name: "arg2",
						Type: reflect.TypeOf(""),
					},
				},
			},
			expectedArg: nil,
		},
	}

	for _, td := range testData {
		require.Equal(t, td.expectedArg, td.m.Option())
	}
}

func TestReader(t *testing.T) {
	testData := []struct {
		m        *generate.Method
		expected bool
	}{
		{
			m: &generate.Method{
				Returns: []reflect.Type{
					reflect.TypeOf(bytes.NewReader([]byte{})), reflect.TypeOf(struct{}{}),
				},
			},
			expected: true,
		},
		{
			m: &generate.Method{
				Returns: []reflect.Type{
					reflect.TypeOf(""),
				},
			},
			expected: false,
		},
	}

	for _, td := range testData {
		require.Equal(t, td.expected, td.m.Reader())
	}
}

func TestWriter(t *testing.T) {
	testData := []struct {
		m            *generate.Method
		expectedName string
	}{
		{
			m: &generate.Method{
				Args: []generate.Arg{
					{
						Name: "arg1",
						Type: reflect.TypeOf(&bytes.Buffer{}),
					},
					{
						Name: "arg2",
						Type: reflect.TypeOf(""),
					},
					{
						Name: "arg3",
						Type: reflect.TypeOf(int(11)),
					},
				},
			},
			expectedName: "arg1",
		},
		{
			m: &generate.Method{
				Args: []generate.Arg{
					{
						Name: "arg2",
						Type: reflect.TypeOf(""),
					},
					{
						Name: "arg3",
						Type: reflect.TypeOf(int(11)),
					},
				},
			},
			expectedName: "",
		},
	}

	for _, td := range testData {
		require.Equal(t, td.expectedName, td.m.Writer())
	}
}

func TestSocketExit(t *testing.T) {
	testData := []struct {
		m        *generate.Method
		expected bool
	}{
		{
			m: &generate.Method{
				Route: generate.Route{
					Method: "SOCKET",
				},
				Returns: []reflect.Type{
					reflect.TypeOf(int(0)), reflect.TypeOf(nil),
				},
			},
			expected: true,
		},
		{
			m: &generate.Method{
				Returns: []reflect.Type{
					reflect.TypeOf(""),
				},
			},
			expected: false,
		},
	}

	for _, td := range testData {
		got, err := td.m.SocketExit()
		require.NoError(t, err)
		require.Equal(t, td.expected, got)
	}
}
