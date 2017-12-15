package aws_test

import (
	"testing"

	"github.com/convox/rack/structs"
	"github.com/convox/rack/test/awsutil"

	"github.com/stretchr/testify/assert"
)

func init() {
}

func TestRegistryAddBlankPassword(t *testing.T) {
	provider := StubAwsProvider(
		cycleRegistryAddRegistry,
	)
	defer provider.Close()

	_, err := provider.RegistryAdd("server", "username", "")

	assert.EqualError(t, err, "password must not be blank", "an error is expected")
}

func TestRegistryAddBlankServer(t *testing.T) {
	provider := StubAwsProvider(
		cycleRegistryAddRegistry,
	)
	defer provider.Close()

	_, err := provider.RegistryAdd("", "username", "password")

	assert.EqualError(t, err, "server must not be blank", "an error is expected")
}

func TestRegistryAddBlankUser(t *testing.T) {
	provider := StubAwsProvider(
		cycleRegistryAddRegistry,
	)
	defer provider.Close()

	_, err := provider.RegistryAdd("server", "", "password")

	assert.EqualError(t, err, "username must not be blank", "an error is expected")
}

func TestRegistryRemove(t *testing.T) {
	provider := StubAwsProvider(
		cycleRegistryHeadRegistry,
		cycleRegistryDeleteRegistry,
	)
	defer provider.Close()

	err := provider.RegistryRemove("r.example.org")

	assert.NoError(t, err)
}

func TestRegistryList(t *testing.T) {
	provider := StubAwsProvider(
		cycleRegistryListRegistries,
		cycleRegistryHeadRegistry,
		cycleRegistryGetRegistry,
		cycleRegistryDecrypt,
	)
	defer provider.Close()

	r, err := provider.RegistryList()

	assert.NoError(t, err)
	assert.EqualValues(t, structs.Registries{
		structs.Registry{
			Server:   "quay.io",
			Username: "ddollar+test",
			Password: "B0IT2U7BZ4VDZUYFM6LFMTJPF8YGKWYBR39AWWPAUKZX6YKZX3SQNBCCQKMX08UF",
		},
	}, r)
}

var cycleRegistryAddRegistry = awsutil.Cycle{
	awsutil.Request{
		Method:     "POST",
		RequestURI: "/registries",
	},
	awsutil.Response{
		StatusCode: 200,
		Body:       "{}",
	},
}

var cycleRegistryListRegistries = awsutil.Cycle{
	awsutil.Request{
		Method:     "GET",
		RequestURI: "/convox-settings?delimiter=%2F&list-type=2&prefix=system%2Fregistries%2F",
	},
	awsutil.Response{
		StatusCode: 200,
		Body: `
			<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
				<Name>convox-settings</Name>
				<Prefix>system/registries/</Prefix>
				<KeyCount>2</KeyCount>
				<MaxKeys>1000</MaxKeys>
				<Delimiter>/</Delimiter>
				<IsTruncated>false</IsTruncated>
				<Contents>
					<Key>system/registries/722e6578616d706c652e6f7267e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855</Key>
					<LastModified>2016-10-04T19:17:48.000Z</LastModified>
					<ETag>&quot;97469e3ca4f6cbec29d79000e1d60054-1&quot;</ETag>
					<Size>161</Size>
					<StorageClass>STANDARD</StorageClass>
				</Contents>
			</ListBucketResult>
		`,
	},
}

var cycleRegistryDeleteRegistry = awsutil.Cycle{
	awsutil.Request{
		Method:     "DELETE",
		RequestURI: "/convox-settings/system/registries/722e6578616d706c652e6f7267e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	},
	awsutil.Response{
		StatusCode: 200,
		Body:       `{}`,
	},
}

var cycleRegistryGetRegistry = awsutil.Cycle{
	awsutil.Request{
		Method:     "GET",
		RequestURI: "/convox-settings/system/registries/722e6578616d706c652e6f7267e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	},
	awsutil.Response{
		StatusCode: 200,
		Body:       `{"c":"cfzlIX7TNG1GAKif6/lRPazbGOOJiiQUyvcaiZiEhedPqUWlp5IoSV8lVYB7iX1U+rRILRKnPYsjN4HCDhZXj5dMmbo6NmS6twkgo0/EIRSvE49lJikHV2GImv2/RV/PpY9Dq1VsHKqDs8jK89qMGTsmDK4C98ziGvfPexSwb68vmzEN3Mw4PoosXbI=","k":"AQEBAHhZfaDM9nag/rJj14qS3jZ+uhrcVvAPT3gOpF4GL4TYAAAAAH4wfAYJKoZIhvcNAQcGoG8wbQIBADBoBgkqhkiG9w0BBwEwHgYJYIZIAWUDBAEuMBEEDO0S9xrGCJKPJdHpDQIBEIA7m12OymJ0sCDdru7RxWOQkbnZtR2XO5WMoFUZW1QL9oU31InZ2Gg+NLqYgT5TgZjz1JhPPXur3kku4CU=","n":"AyExwRSvnP1WWaPmCvGy6+AcpCMIWanJ"}`,
	},
}

var cycleRegistryHeadRegistry = awsutil.Cycle{
	awsutil.Request{
		Method:     "HEAD",
		RequestURI: "/convox-settings/system/registries/722e6578616d706c652e6f7267e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	},
	awsutil.Response{
		StatusCode: 200,
		Body: `{
			"server": "foo",
			"username": "bar",
			"password": "baz"
		}`,
	},
}

var cycleRegistryDecrypt = awsutil.Cycle{
	awsutil.Request{
		RequestURI: "/",
		Operation:  "TrentService.Decrypt",
		Body: `{
			"CiphertextBlob": "AQEBAHhZfaDM9nag/rJj14qS3jZ+uhrcVvAPT3gOpF4GL4TYAAAAAH4wfAYJKoZIhvcNAQcGoG8wbQIBADBoBgkqhkiG9w0BBwEwHgYJYIZIAWUDBAEuMBEEDO0S9xrGCJKPJdHpDQIBEIA7m12OymJ0sCDdru7RxWOQkbnZtR2XO5WMoFUZW1QL9oU31InZ2Gg+NLqYgT5TgZjz1JhPPXur3kku4CU="
		}`,
	},
	awsutil.Response{
		StatusCode: 200,
		Body: `{
			"Plaintext": "rOoFLwHSrzecza1KGCFPWxnSjL3gROW0XFxbUDAsBiQ="
		}`,
	},
}
