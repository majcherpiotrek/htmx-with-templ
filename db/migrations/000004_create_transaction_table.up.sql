CREATE TABLE IF NOT EXISTS transaction(
	id bigserial PRIMARY KEY,
	plaid_transaction_id VARCHAR(255) not null,
	bank_account_id int not null,
	amount NUMERIC(15,3) not null,
	currency VARCHAR(255) not null,
	date_authorized DATE not null,
	date_time_authorized TIMESTAMP WITH TIME ZONE,
	date_posted DATE not null,
	date_time_posted TIMESTAMP WITH TIME ZONE,
	next_cursor VARCHAR(255),


	FOREIGN KEY(bank_account_id) REFERENCES bank_account(id)
);
