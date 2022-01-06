package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/mail"

	"github.com/wisdommatt/nekom-assessment/proto/customer"
	"github.com/wisdommatt/nekom-assessment/proto/orders"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

type customerOrdersResponse struct {
	Status  string                  `json:"status"`
	Message string                  `json:"message"`
	Orders  []*orders.OrderResponse `json:"orders"`
}

var (
	hostAddress = "stage.nekom.com:443"
	httpPort    = "8080"
)

func main() {
	opt := grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{}))
	grpcConn, err := grpc.Dial(hostAddress, opt)
	if err != nil {
		log.Fatal("grpc conn error: ", err.Error())
	}
	customerClient := customer.NewCustomerConnectorClient(grpcConn)
	ordersClient := orders.NewOrdersConnectorClient(grpcConn)

	serveMux := http.NewServeMux()
	serveMux.Handle("/orders/", handleGetCustomerOrders(customerClient, ordersClient))
	log.Printf("app running on http://localhost:%s", httpPort)
	log.Fatal(http.ListenAndServe(":"+httpPort, serveMux))
}

// handleGetCustomerOrders is the http route handler for retrieving
// all customer's order filtering by the customer's email.
func handleGetCustomerOrders(
	customerClient customer.CustomerConnectorClient,
	ordersClient orders.OrdersConnectorClient,
) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		email := r.URL.Query().Get("email")
		if _, err := mail.ParseAddress(email); err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(rw).Encode(customerOrdersResponse{
				Status:  "error",
				Message: fmt.Sprintf("invalid email address: '%s'", email),
			})
			return
		}
		metaData := metadata.New(map[string]string{
			"token":      "storer@2020!",
			"clientuuid": "fdfd497828effdfda2b049a27849a2d4",
		})
		ctx := metadata.NewOutgoingContext(r.Context(), metaData)
		customerResponse, err := customerClient.GetCustomerByEmail(ctx, &customer.Email{Email: email})
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(rw).Encode(customerOrdersResponse{
				Status:  "error",
				Message: err.Error(),
			})
			return
		}
		if len(customerResponse.Customers) == 0 {
			rw.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(rw).Encode(customerOrdersResponse{
				Status:  "error",
				Message: "a customer with this email does not exist",
			})
			return
		}
		customer := customerResponse.Customers[0]
		ordersResponse, err := ordersClient.GetOrdersForCustomer(ctx, &orders.CustomerId{
			Customerid: int32(customer.Customerid),
		})
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(rw).Encode(customerOrdersResponse{
				Status:  "error",
				Message: err.Error(),
			})
			return
		}
		response := customerOrdersResponse{
			Status:  "success",
			Message: "Orders retrieved successfully",
			Orders:  ordersResponse.Orders,
		}
		json.NewEncoder(rw).Encode(response)
	})
}
