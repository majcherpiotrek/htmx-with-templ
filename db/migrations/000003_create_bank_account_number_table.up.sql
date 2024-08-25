CREATE TABLE IF NOT EXISTS bank_account_number(
	id serial PRIMARY KEY,
	bank_account_id INTEGER not null,
	account_number_type VARCHAR(255) not null,
	account VARCHAR(255),
	routing VARCHAR(255),
	wire_routing VARCHAR(255),
	institution VARCHAR(255),
	branch VARCHAR(255),
	bic VARCHAR(255),
	iban VARCHAR(255),
	sort_code VARCHAR(255),

	FOREIGN KEY(bank_account_id) REFERENCES bank_account(id) 
);
