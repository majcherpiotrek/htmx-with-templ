CREATE TABLE IF NOT EXISTS bank_connection(
	id serial PRIMARY KEY,
	plaid_item_id VARCHAR(50) unique not null,
	access_token VARCHAR(255) not null
);
