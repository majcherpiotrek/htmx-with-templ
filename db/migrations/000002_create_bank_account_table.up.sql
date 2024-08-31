CREATE TABLE IF NOT EXISTS bank_account(
	id serial PRIMARY KEY,
	plaid_account_id VARCHAR(255) unique not null,
	bank_connection_id INTEGER not null,
	name VARCHAR(255) not null,
	mask VARCHAR(255) not null,
	account_type VARCHAR(255) not null,
	current_balance NUMERIC(15,3),
	available_balance NUMERIC(15,3),
	currency VARCHAR(255) not null,

	FOREIGN KEY(bank_connection_id) REFERENCES bank_connection(id) 
);
