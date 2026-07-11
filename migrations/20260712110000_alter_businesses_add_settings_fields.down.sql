ALTER TABLE businesses
    DROP COLUMN IF EXISTS quantity_precision,
    DROP COLUMN IF EXISTS currency_precision,
    DROP COLUMN IF EXISTS time_format,
    DROP COLUMN IF EXISTS date_format,
    DROP COLUMN IF EXISTS transaction_edit_days,
    DROP COLUMN IF EXISTS stock_accounting_method,
    DROP COLUMN IF EXISTS financial_year_start_month,
    DROP COLUMN IF EXISTS logo_url,
    DROP COLUMN IF EXISTS timezone,
    DROP COLUMN IF EXISTS currency_symbol_placement,
    DROP COLUMN IF EXISTS currency,
    DROP COLUMN IF EXISTS default_profit_percentage,
    DROP COLUMN IF EXISTS start_date;
