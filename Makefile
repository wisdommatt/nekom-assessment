gen-proto:
	protoc ordersconnector_plain.proto --go-grpc_out=. --go_out=.
	protoc apicustomer_plain.proto --go-grpc_out=. --go_out=.