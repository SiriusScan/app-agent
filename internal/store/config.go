package store

// Config holds configuration for the store client
type Config struct {
	// Address is the store server address
	Address string

	// Username for authentication
	Username string

	// Password for authentication
	Password string

	// Database name or number
	Database string

	// ConnectionTimeout specifies the timeout for connecting to the store
	ConnectionTimeout int

	// OperationTimeout specifies the timeout for store operations
	OperationTimeout int
}
