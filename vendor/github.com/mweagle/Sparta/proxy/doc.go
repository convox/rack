package proxy

//go:generate mkdir -pv ../resources/provision/proto
//go:generate protoc --js_out=import_style=commonjs,binary:../resources/provision/proto proxy.proto
//go:generate protoc --python_out=../resources/provision/proto proxy.proto
//go:generate protoc --go_out=plugins=grpc:./ proxy.proto
