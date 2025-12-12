package model

// Минималистичная и удобная модель ответа от сервера, всегда использую
type Response struct {
	Success bool   `json:"success" example:"true"`
	Error   string `json:"error,omitempty" example:"Error message"`
	Data    any    `json:"data,omitempty"`
}

// Модель для обновления баланса, все поля нужные, также есть примеры и
// прописаны базовые требования к полям тела запроса
type UpdateBalance struct {
	WalletId      string `json:"valletId" example:"550e8400-e29b-41d4-a716-446655440000" binding:"required"`
	OperationType string `json:"operationType" example:"DEPOSIT" enums:"DEPOSIT,WITHDRAW" binding:"required"`
	Amount        int64  `json:"amount" example:"1000" binding:"required,gt=0"`
}
