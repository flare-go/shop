-- 刪除觸發器
DROP TRIGGER IF EXISTS update_cart_trigger ON cart_items;
DROP TRIGGER IF EXISTS update_order_trigger ON order_items;
DROP TRIGGER IF EXISTS reserve_stock_trigger ON cart_items;
DROP TRIGGER IF EXISTS release_stock_trigger ON cart_items;

-- 刪除觸發器函數
DROP FUNCTION IF EXISTS update_cart;
DROP FUNCTION IF EXISTS update_order;
DROP FUNCTION IF EXISTS reserve_stock;
DROP FUNCTION IF EXISTS release_stock;

-- 刪除索引
DROP INDEX IF EXISTS idx_carts_customer_id;
DROP INDEX IF EXISTS idx_carts_status;
DROP INDEX IF EXISTS idx_cart_items_cart_id;
DROP INDEX IF EXISTS idx_cart_items_product_id;
DROP INDEX IF EXISTS idx_orders_customer_id;
DROP INDEX IF EXISTS idx_orders_status;
DROP INDEX IF EXISTS idx_orders_payment_intent_id;
DROP INDEX IF EXISTS idx_order_items_order_id;
DROP INDEX IF EXISTS idx_order_items_product_id;
DROP INDEX IF EXISTS idx_product_categories_product_id;
DROP INDEX IF EXISTS idx_product_categories_category_id;
DROP INDEX IF EXISTS idx_stocks_product_id;
DROP INDEX IF EXISTS idx_stock_movements_stock_id;
DROP INDEX IF EXISTS idx_stock_movements_reference;

-- 刪除表格
DROP TABLE IF EXISTS product_categories;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS cart_items;
DROP TABLE IF EXISTS carts;
DROP TABLE IF EXISTS stock_movements CASCADE ;
DROP TABLE IF EXISTS stocks CASCADE;


DROP TYPE IF EXISTS cart_status;
DROP TYPE IF EXISTS order_status;
DROP TYPE IF EXISTS stock_movement_reference_type;
DROP TYPE IF EXISTS stock_movement_type;
