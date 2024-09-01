package transactions

type TransactionRepository interface {
	ListAllForAccount(bankAccountID int) ([]DbTransaction, error)
	ListAll() ([]DbTransaction, error)
	SaveAll([]DbTransactionWriteModel) ([]DbTransaction, error)
}
