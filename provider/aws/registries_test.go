package aws_test

import (
	"testing"

	"github.com/convox/rack/api/awsutil"
	"github.com/convox/rack/api/structs"

	"github.com/stretchr/testify/assert"
)

func init() {
}

func TestRegistryDelete(t *testing.T) {
	provider := StubAwsProvider(
		cycleRegistryGetRegistry,
		cycleRegistryGetRegistry,
		cycleRegistryDeleteRegistry,
	)
	defer provider.Close()

	err := provider.RegistryDelete("r.example.org")

	assert.Nil(t, err)
}

func TestRegistryList(t *testing.T) {
	provider := StubAwsProvider(
		cycleRegistryGetRackEnv,
		cycleRegistryListRegistries,
		cycleRegistryGetRegistry,
	)
	defer provider.Close()

	r, err := provider.RegistryList()

	assert.Nil(t, err)
	assert.EqualValues(t, structs.Registries{
		structs.Registry{
			Server:   "foo",
			Username: "bar",
			Password: "baz",
		},
	}, r)
}

var cycleRegistryGetRackEnv = awsutil.Cycle{
	awsutil.Request{
		RequestURI: "/convox-settings/env",
		Operation:  "",
		Body:       "",
	},
	awsutil.Response{
		StatusCode: 200,
		Body:       "{}",
	},
}

var cycleRegistryListRegistries = awsutil.Cycle{
	awsutil.Request{
		RequestURI: "/convox-settings?delimiter=%2F&list-type=2&prefix=system%2Fregistries%2F",
		Operation:  "",
		Body:       "",
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
		Operation:  "",
		Body:       "",
	},
	awsutil.Response{
		StatusCode: 200,
		Body:       `{}`,
	},
}

var cycleRegistryGetRegistry = awsutil.Cycle{
	awsutil.Request{
		RequestURI: "/convox-settings/system/registries/722e6578616d706c652e6f7267e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		Operation:  "",
		Body:       "",
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
