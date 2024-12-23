package handler

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/tittuvarghese/ss-go-core/logger"
	"github.com/tittuvarghese/ss-go-order-management-service/core/database"
	"github.com/tittuvarghese/ss-go-order-management-service/models"
	"github.com/tittuvarghese/ss-go-order-management-service/proto"
	"github.com/tittuvarghese/ss-go-order-management-service/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
)

type Server struct {
	proto.UnimplementedOrderServiceServer
	GrpcServer  *grpc.Server
	RdbInstance *database.RelationalDatabase
}

var log = logger.NewLogger("order-management-service")

func NewGrpcServer() *Server {
	return &Server{GrpcServer: grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))}
}

func (s *Server) Run(port string) {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Error("Failed to listen", err)
	}

	proto.RegisterOrderServiceServer(s.GrpcServer, s)

	// Register reflection service on gRPC server
	reflection.Register(s.GrpcServer)
	log.Info("GRPC server is listening on port " + port)
	if err := s.GrpcServer.Serve(lis); err != nil {
		log.Error("failed to serve", err)
	}
}

func (s *Server) mustEmbedUnimplementedAuthServiceServer() {
	log.Error("implement me", nil)
}

func (s *Server) CreateOrder(ctx context.Context, req *proto.CreateOrderRequest) (*proto.CreateOrderResponse, error) {
	var order models.Order
	var totalPrice float64

	for _, item := range req.Items {
		totalPrice += item.Price * float64(item.Quantity)
		orderedItem := models.Item{
			ProductID: item.ProductId,
			Quantity:  item.Quantity,
			Price:     item.Price,
		}
		order.Items = append(order.Items, orderedItem)
	}
	order.TotalPrice = totalPrice

	buyerId, err := uuid.Parse(req.CustomerId)
	if err != nil {
		return &proto.CreateOrderResponse{
			Message: "Unable to parse buyer id",
		}, err
	}
	order.CustomerID = buyerId
	order.Address = models.Address{
		AddressLine1: req.Address.AddressLine1,
		AddressLine2: req.Address.AddressLine2,
		City:         req.Address.City,
		State:        req.Address.State,
		Zip:          req.Address.Zip,
		Country:      req.Address.Country,
	}
	order.Phone = req.Phone

	err = service.CreateOrder(order, s.RdbInstance)

	if err != nil {
		return &proto.CreateOrderResponse{
			Message: "Failed to create the order. error: " + err.Error(),
		}, err
	}

	// Return the created product
	return &proto.CreateOrderResponse{Message: "Successfully created the order"}, nil
}

func (s *Server) GetOrders(ctx context.Context, req *proto.GetOrdersRequest) (*proto.GetOrdersResponse, error) {
	// Parse the customer ID
	buyerId, err := uuid.Parse(req.CustomerId)
	if err != nil {
		return &proto.GetOrdersResponse{
			Message: "Unable to parse buyer id",
		}, err
	}

	// Get the orders for the customer
	orders, err := service.GetOrders(buyerId, s.RdbInstance)
	if err != nil {
		return nil, err
	}

	// Create the response to be returned
	var response []*proto.Order

	// Iterate through orders and build the response
	for _, order := range *orders {
		ord := &proto.Order{
			OrderId: order.OrderID,
			Items:   GetItemsFromOrder(order.Items), // Get items from order
			Address: &proto.Address{
				AddressLine1: order.Address.AddressLine1,
				AddressLine2: order.Address.AddressLine2,
				City:         order.Address.City,
				State:        order.Address.State,
				Zip:          order.Address.Zip,
				Country:      order.Address.Country,
			},
			Phone: order.Phone,
		}
		// Append the order to the response slice
		response = append(response, ord)
	}

	// Return the response with the list of orders
	return &proto.GetOrdersResponse{
		Orders: response,
	}, nil
}

func (s *Server) GetOrder(ctx context.Context, req *proto.GetOrderRequest) (*proto.GetOrderResponse, error) {
	// Parse the customer ID
	buyerId, err := uuid.Parse(req.CustomerId)
	if err != nil {
		return &proto.GetOrderResponse{
			Message: "Unable to parse buyer id",
		}, err
	}

	// Get the orders for the customer
	orderResult, err := service.GetOrder(buyerId, req.OrderId, s.RdbInstance)
	if err != nil {
		return nil, err
	}

	if len(orderResult) <= 0 {
		log.Error("no order found", nil)
		return &proto.GetOrderResponse{
			Message: "No order found",
		}, fmt.Errorf("no order found")
	}

	order := orderResult[0]

	// Iterate through orders and build the response
	response := &proto.Order{
		OrderId: order.OrderID,
		Status:  string(order.Status),
		Items:   GetItemsFromOrder(order.Items), // Get items from order
		Address: &proto.Address{
			AddressLine1: order.Address.AddressLine1,
			AddressLine2: order.Address.AddressLine2,
			City:         order.Address.City,
			State:        order.Address.State,
			Zip:          order.Address.Zip,
			Country:      order.Address.Country,
		},
		Phone: order.Phone,
	}

	// Return the response with the list of orders
	return &proto.GetOrderResponse{
		Order:   response,
		Message: "Successfully retrieved the order",
	}, nil
}

func GetItemsFromOrder(order []models.Item) []*proto.OrderItem {
	// Initialize the slice with capacity equal to len(order) for efficiency
	items := make([]*proto.OrderItem, 0, len(order))

	// Populate the items slice with pointers to OrderItems
	for _, item := range order {
		ord := &proto.OrderItem{
			Quantity:  item.Quantity,
			Price:     item.Price,
			ProductId: item.ProductID,
		}
		// Append the pointer to the slice
		items = append(items, ord)
	}

	// Return the slice directly
	return items
}

func (s *Server) UpdateOrderStatus(ctx context.Context, req *proto.UpdateOrderStatusRequest) (*proto.UpdateOrderStatusResponse, error) {
	// Parse the customer ID
	buyerId, err := uuid.Parse(req.CustomerId)
	if err != nil {
		return &proto.UpdateOrderStatusResponse{
			Message: "Unable to parse buyer id",
		}, err
	}

	fmt.Println(buyerId)
	fmt.Println(req.OrderId)
	fmt.Println(req.Status)

	// Get the orders for the customer
	orderResult, err := service.GetOrder(buyerId, req.OrderId, s.RdbInstance)
	if err != nil {
		return nil, err
	}

	if len(orderResult) <= 0 {
		log.Error("no order found", nil)
		return &proto.UpdateOrderStatusResponse{
			Message: "No order found",
		}, fmt.Errorf("no order found")
	}

	order := orderResult[0]

	if order.Status != models.OrderStatus(req.Status) {
		order.Status = models.OrderStatus(req.Status)
	}

	err = service.UpdateOrder(order, s.RdbInstance)
	if err != nil {
		return &proto.UpdateOrderStatusResponse{
			Message: "Failed to update the order status. error: " + err.Error(),
		}, err
	}

	// Return the created product
	return &proto.UpdateOrderStatusResponse{Message: "Successfully updated the order status"}, nil
}
