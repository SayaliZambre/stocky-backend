-- Insert common Indian stocks
INSERT INTO stocks (symbol, name, exchange) VALUES 
('RELIANCE', 'Reliance Industries Limited', 'NSE'),
('TCS', 'Tata Consultancy Services Limited', 'NSE'),
('INFY', 'Infosys Limited', 'NSE'),
('HDFCBANK', 'HDFC Bank Limited', 'NSE'),
('ICICIBANK', 'ICICI Bank Limited', 'NSE'),
('HINDUNILVR', 'Hindustan Unilever Limited', 'NSE'),
('ITC', 'ITC Limited', 'NSE'),
('SBIN', 'State Bank of India', 'NSE'),
('BHARTIARTL', 'Bharti Airtel Limited', 'NSE'),
('KOTAKBANK', 'Kotak Mahindra Bank Limited', 'NSE')
ON CONFLICT (symbol) DO NOTHING;
