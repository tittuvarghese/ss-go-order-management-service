package service

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/tittuvarghese/ss-go-order-management-service/core/database"
	"github.com/tittuvarghese/ss-go-order-management-service/models"
)

func CreateOrder(order models.Order, storage *database.RelationalDatabase) error {

	var transaction = database.DbTxn

	// Building transaction
	// 1. Order creation
	orderCreation := database.DbOps
	orderCreation.Model = &order
	orderCreation.Command = database.CreateCommand

	transaction.Operations = append(transaction.Operations, orderCreation)

	// Update Quantity
	for _, item := range order.Items {
		condition := map[string]interface{}{"id": item.ProductID}
		var queryUpdate = database.DbOps
		queryUpdate.Model = &models.Product{}
		queryUpdate.Condition = condition
		queryUpdate.Command = database.UpdateCommand

		var queryExpr = database.DbExpr
		queryExpr.Column = "quantity"
		queryExpr.Value = storage.Instance.BuildExpr("quantity - ?", item.Quantity)
		queryUpdate.Expr = queryExpr
		transaction.Operations = append(transaction.Operations, queryUpdate)
	}

	err := storage.Instance.Transaction(transaction)

	if err != nil {
		return err
	}
	return nil

}

func GetOrders(customerId uuid.UUID, storage *database.RelationalDatabase) (*[]models.Order, error) {
	var orders []models.Order
	condition := map[string]interface{}{"customer_id": customerId}
	tables := []string{"Items", "Address"}

	// Pass a slice of User to QueryByCondition
	res, err := storage.Instance.QueryByCondition(&orders, condition, tables...)
	if err != nil {
		return nil, err
	}

	// Check if the result contains any user
	if len(res) == 0 {
		return nil, fmt.Errorf("no orders found")
	}

	foundOrder, _ := res[0].(*[]models.Order)

	if len(*foundOrder) == 0 {
		return nil, fmt.Errorf("no orders found")
	}
	return foundOrder, nil
}

func GetOrder(customerId uuid.UUID, orderId string, storage *database.RelationalDatabase) ([]models.Order, error) {
	var orders []models.Order
	condition := map[string]interface{}{"customer_id": customerId, "order_id": orderId}

	tables := []string{"Items", "Address"}

	// Query the database with the given condition
	res, err := storage.Instance.QueryByCondition(&orders, condition, tables...)
	if err != nil {
		return []models.Order{}, err
	}

	// Check if the result contains any products
	if len(res) <= 0 {
		return []models.Order{}, fmt.Errorf("order not found")
	}

	foundOrder, ok := res[0].(*[]models.Order) // Type assertion to pointer of models.Product
	if !ok {
		return []models.Order{}, fmt.Errorf("type assertion failed")
	}

	return *foundOrder, nil
}
func UpdateOrder(order models.Order, storage *database.RelationalDatabase) error {
	err := storage.Instance.Update(&order)
	if err != nil {
		return err
	}
	return nil
}
